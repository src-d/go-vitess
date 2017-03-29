package ldapauthserver

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	log "github.com/golang/glog"
	"github.com/youtube/vitess/go/mysqlconn"
	"github.com/youtube/vitess/go/netutil"
	querypb "github.com/youtube/vitess/go/vt/proto/query"
	"github.com/youtube/vitess/go/vt/servenv/grpcutils"
	"gopkg.in/ldap.v2"
)

var (
	ldapAuthConfigFile   = flag.String("mysql_ldap_auth_config_file", "", "JSON File from which to read LDAP server config.")
	ldapAuthConfigString = flag.String("mysql_ldap_auth_config_string", "", "JSON representation of LDAP server config.")
)

// AuthServerLdap implements AuthServer with an LDAP backend
type AuthServerLdap struct {
	Client
	ServerConfig
	User           string
	Password       string
	GroupQuery     string
	UserDnPattern  string
	RefreshSeconds time.Duration
}

// Init is public so it can be called from plugin_auth_ldap.go (go/cmd/vtgate)
func Init() {
	if *ldapAuthConfigFile == "" && *ldapAuthConfigString == "" {
		log.Infof("Not configuring AuthServerLdap because mysql_ldap_auth_config_file and mysql_ldap_auth_config_string are empty")
		return
	}
	if *ldapAuthConfigFile != "" && *ldapAuthConfigString != "" {
		log.Infof("Both mysql_ldap_auth_config_file and mysql_ldap_auth_config_string are non-empty, can only use one.")
		return
	}
	ldapAuthServer := &AuthServerLdap{Client: &ClientImpl{}, ServerConfig: ServerConfig{}}

	data := []byte(*ldapAuthConfigString)
	if *ldapAuthConfigFile != "" {
		var err error
		data, err = ioutil.ReadFile(*ldapAuthConfigFile)
		if err != nil {
			log.Fatalf("Failed to read mysql_ldap_auth_config_file: %v", err)
		}
	}
	if err := json.Unmarshal(data, ldapAuthServer); err != nil {
		log.Fatalf("Error parsing AuthServerLdap config: %v", err)
	}
	mysqlconn.RegisterAuthServerImpl("ldap", ldapAuthServer)
}

// UseClearText is always true for AuthServerLdap
func (asl *AuthServerLdap) UseClearText() bool {
	return true
}

// Salt is unimplemented for AuthServerLdap
func (asl *AuthServerLdap) Salt() ([]byte, error) {
	panic("unimplemented")
}

// ValidateHash is unimplemented for AuthServerLdap
func (asl *AuthServerLdap) ValidateHash(salt []byte, user string, authResponse []byte) (mysqlconn.Getter, error) {
	panic("unimplemented")
}

// ValidateClearText connects to the LDAP server over TLS
// and attempts to bind as that user with the supplied password.
// It returns the supplied username.
func (asl *AuthServerLdap) ValidateClearText(username, password string) (mysqlconn.Getter, error) {
	err := asl.Client.Connect("tcp", &asl.ServerConfig)
	if err != nil {
		return nil, err
	}
	defer asl.Client.Close()
	err = asl.Client.Bind(fmt.Sprintf(asl.UserDnPattern, username), password)
	if err != nil {
		return nil, err
	}
	groups, err := asl.getGroups(username)
	if err != nil {
		return nil, err
	}
	return &LdapUserData{asl: asl, groups: groups, username: username, lastUpdated: time.Now(), updating: false}, nil
}

//this needs to be passed an already connected client...should check for this
func (asl *AuthServerLdap) getGroups(username string) ([]string, error) {
	err := asl.Client.Bind(asl.User, asl.Password)
	if err != nil {
		return nil, err
	}
	req := ldap.NewSearchRequest(
		asl.GroupQuery,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(memberUid=%s)", username),
		[]string{"cn"},
		nil,
	)
	res, err := asl.Client.Search(req)
	if err != nil {
		return nil, err
	}
	var groups []string
	for _, entry := range res.Entries {
		for _, attr := range entry.Attributes {
			groups = append(groups, attr.Values[0])
		}
	}
	return groups, nil
}

// LdapUserData holds username and LDAP groups as well as enough data to
// intelligently update itself.
type LdapUserData struct {
	asl         *AuthServerLdap
	groups      []string
	username    string
	lastUpdated time.Time
	updating    bool
	sync.Mutex
}

func (lud *LdapUserData) update() {
	lud.Lock()
	if lud.updating {
		lud.Unlock()
		return
	}
	lud.updating = true
	lud.Unlock()
	err := lud.asl.Client.Connect("tcp", &lud.asl.ServerConfig)
	if err != nil {
		log.Errorf("Error updating LDAP user data: %v", err)
		return
	}
	defer lud.asl.Client.Close() //after the error check
	groups, err := lud.asl.getGroups(lud.username)
	if err != nil {
		log.Errorf("Error updating LDAP user data: %v", err)
		return
	}
	lud.Lock()
	lud.groups = groups
	lud.lastUpdated = time.Now()
	lud.updating = false
	lud.Unlock()
}

// Get returns wrapped username and LDAP groups and possibly updates the cache
func (lud *LdapUserData) Get() *querypb.VTGateCallerID {
	if time.Since(lud.lastUpdated) > lud.asl.RefreshSeconds*time.Second {
		go lud.update()
	}
	return &querypb.VTGateCallerID{Username: lud.username, Groups: lud.groups}
}

// ServerConfig holds the config for and LDAP server
// * include port in ldapServer, "ldap.example.com:386"
type ServerConfig struct {
	LdapServer string
	LdapCert   string
	LdapKey    string
	LdapCA     string
}

// Client provides an interface we can mock
type Client interface {
	Connect(network string, config *ServerConfig) error
	Close()
	Bind(string, string) error
	Search(*ldap.SearchRequest) (*ldap.SearchResult, error)
}

// ClientImpl is the real implementation of LdapClient
type ClientImpl struct {
	*ldap.Conn
}

// Connect calls ldap.Dial and then upgrades the connection to TLS
// This must be called before any other methods
func (lci *ClientImpl) Connect(network string, config *ServerConfig) error {
	conn, err := ldap.Dial(network, config.LdapServer)
	lci.Conn = conn
	// Reconnect with TLS ... why don't we simply DialTLS directly?
	serverName, _, err := netutil.SplitHostPort(config.LdapServer)
	if err != nil {
		return err
	}
	tlsConfig, err := grpcutils.TLSClientConfig(config.LdapCert, config.LdapKey, config.LdapCA, serverName)
	if err != nil {
		return err
	}
	err = conn.StartTLS(tlsConfig)
	if err != nil {
		return err
	}
	return nil
}
