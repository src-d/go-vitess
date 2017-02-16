// Code generated by protoc-gen-go.
// source: throttlerdata.proto
// DO NOT EDIT!

/*
Package throttlerdata is a generated protocol buffer package.

It is generated from these files:
	throttlerdata.proto

It has these top-level messages:
	MaxRatesRequest
	MaxRatesResponse
	SetMaxRateRequest
	SetMaxRateResponse
	Configuration
	GetConfigurationRequest
	GetConfigurationResponse
	UpdateConfigurationRequest
	UpdateConfigurationResponse
	ResetConfigurationRequest
	ResetConfigurationResponse
*/
package throttlerdata

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// MaxRatesRequest is the payload for the MaxRates RPC.
type MaxRatesRequest struct {
}

func (m *MaxRatesRequest) Reset()                    { *m = MaxRatesRequest{} }
func (m *MaxRatesRequest) String() string            { return proto.CompactTextString(m) }
func (*MaxRatesRequest) ProtoMessage()               {}
func (*MaxRatesRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

// MaxRatesResponse is returned by the MaxRates RPC.
type MaxRatesResponse struct {
	// max_rates returns the max rate for each throttler. It's keyed by the
	// throttler name.
	Rates map[string]int64 `protobuf:"bytes,1,rep,name=rates" json:"rates,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"varint,2,opt,name=value"`
}

func (m *MaxRatesResponse) Reset()                    { *m = MaxRatesResponse{} }
func (m *MaxRatesResponse) String() string            { return proto.CompactTextString(m) }
func (*MaxRatesResponse) ProtoMessage()               {}
func (*MaxRatesResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *MaxRatesResponse) GetRates() map[string]int64 {
	if m != nil {
		return m.Rates
	}
	return nil
}

// SetMaxRateRequest is the payload for the SetMaxRate RPC.
type SetMaxRateRequest struct {
	Rate int64 `protobuf:"varint,1,opt,name=rate" json:"rate,omitempty"`
}

func (m *SetMaxRateRequest) Reset()                    { *m = SetMaxRateRequest{} }
func (m *SetMaxRateRequest) String() string            { return proto.CompactTextString(m) }
func (*SetMaxRateRequest) ProtoMessage()               {}
func (*SetMaxRateRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

// SetMaxRateResponse is returned by the SetMaxRate RPC.
type SetMaxRateResponse struct {
	// names is the list of throttler names which were updated.
	Names []string `protobuf:"bytes,1,rep,name=names" json:"names,omitempty"`
}

func (m *SetMaxRateResponse) Reset()                    { *m = SetMaxRateResponse{} }
func (m *SetMaxRateResponse) String() string            { return proto.CompactTextString(m) }
func (*SetMaxRateResponse) ProtoMessage()               {}
func (*SetMaxRateResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

// Configuration holds the configuration parameters for the
// MaxReplicationLagModule which adaptively adjusts the throttling rate based on
// the observed replication lag across all replicas.
type Configuration struct {
	// target_replication_lag_sec is the replication lag (in seconds) the
	// MaxReplicationLagModule tries to aim for.
	// If it is within the target, it tries to increase the throttler
	// rate, otherwise it will lower it based on an educated guess of the
	// slave throughput.
	TargetReplicationLagSec int64 `protobuf:"varint,1,opt,name=target_replication_lag_sec,json=targetReplicationLagSec" json:"target_replication_lag_sec,omitempty"`
	// max_replication_lag_sec is meant as a last resort.
	// By default, the module tries to find out the system maximum capacity while
	// trying to keep the replication lag around "target_replication_lag_sec".
	// Usually, we'll wait min_duration_between_(increases|decreases)_sec to see
	// the effect of a throttler rate change on the replication lag.
	// But if the lag goes above this field's value we will go into an "emergency"
	// state and throttle more aggressively (see "emergency_decrease" below).
	// This is the only way to ensure that the system will recover.
	MaxReplicationLagSec int64 `protobuf:"varint,2,opt,name=max_replication_lag_sec,json=maxReplicationLagSec" json:"max_replication_lag_sec,omitempty"`
	// initial_rate is the rate at which the module will start.
	InitialRate int64 `protobuf:"varint,3,opt,name=initial_rate,json=initialRate" json:"initial_rate,omitempty"`
	// max_increase defines by how much we will increase the rate
	// e.g. 0.05 increases the rate by 5% while 1.0 by 100%.
	// Note that any increase will let the system wait for at least
	// (1 / MaxIncrease) seconds. If we wait for shorter periods of time, we
	// won't notice if the rate increase also increases the replication lag.
	// (If the system was already at its maximum capacity (e.g. 1k QPS) and we
	// increase the rate by e.g. 5% to 1050 QPS, it will take 20 seconds until
	// 1000 extra queries are buffered and the lag increases by 1 second.)
	MaxIncrease float64 `protobuf:"fixed64,4,opt,name=max_increase,json=maxIncrease" json:"max_increase,omitempty"`
	// emergency_decrease defines by how much we will decrease the current rate
	// if the observed replication lag is above "max_replication_lag_sec".
	// E.g. 0.50 decreases the current rate by 50%.
	EmergencyDecrease float64 `protobuf:"fixed64,5,opt,name=emergency_decrease,json=emergencyDecrease" json:"emergency_decrease,omitempty"`
	// min_duration_between_increases_sec specifies how long we'll wait at least
	// for the last rate increase to have an effect on the system.
	MinDurationBetweenIncreasesSec int64 `protobuf:"varint,6,opt,name=min_duration_between_increases_sec,json=minDurationBetweenIncreasesSec" json:"min_duration_between_increases_sec,omitempty"`
	// max_duration_between_increases_sec specifies how long we'll wait at most
	// for the last rate increase to have an effect on the system.
	MaxDurationBetweenIncreasesSec int64 `protobuf:"varint,7,opt,name=max_duration_between_increases_sec,json=maxDurationBetweenIncreasesSec" json:"max_duration_between_increases_sec,omitempty"`
	// min_duration_between_decreases_sec specifies how long we'll wait at least
	// for the last rate decrease to have an effect on the system.
	MinDurationBetweenDecreasesSec int64 `protobuf:"varint,8,opt,name=min_duration_between_decreases_sec,json=minDurationBetweenDecreasesSec" json:"min_duration_between_decreases_sec,omitempty"`
	// spread_backlog_across_sec is used when we set the throttler rate after
	// we guessed the rate of a slave and determined its backlog.
	// For example, at a guessed rate of 100 QPS and a lag of 10s, the replica has
	// a backlog of 1000 queries.
	// When we set the new, decreased throttler rate, we factor in how long it
	// will take the slave to go through the backlog (in addition to new
	// requests). This field specifies over which timespan we plan to spread this.
	// For example, for a backlog of 1000 queries spread over 5s means that we
	// have to further reduce the rate by 200 QPS or the backlog will not be
	// processed within the 5 seconds.
	SpreadBacklogAcrossSec int64 `protobuf:"varint,9,opt,name=spread_backlog_across_sec,json=spreadBacklogAcrossSec" json:"spread_backlog_across_sec,omitempty"`
	// ignore_n_slowest_replicas will ignore replication lag updates from the
	// N slowest REPLICA tablets. Under certain circumstances, replicas are still
	// considered e.g. a) if the lag is at most max_replication_lag_sec, b) there
	// are less than N+1 replicas or c) the lag increased on each replica such
	// that all replicas were ignored in a row.
	IgnoreNSlowestReplicas int32 `protobuf:"varint,10,opt,name=ignore_n_slowest_replicas,json=ignoreNSlowestReplicas" json:"ignore_n_slowest_replicas,omitempty"`
	// ignore_n_slowest_rdonlys does the same thing as ignore_n_slowest_replicas
	// but for RDONLY tablets. Note that these two settings are independent.
	IgnoreNSlowestRdonlys int32 `protobuf:"varint,11,opt,name=ignore_n_slowest_rdonlys,json=ignoreNSlowestRdonlys" json:"ignore_n_slowest_rdonlys,omitempty"`
	// age_bad_rate_after_sec is the duration after which an unchanged bad rate
	// will "age out" and increase by "bad_rate_increase".
	// Bad rates are tracked by the code in memory.go and serve as an upper bound
	// for future rate changes. This ensures that the adaptive throttler does not
	// try known too high (bad) rates over and over again.
	// To avoid that temporary degradations permanently reduce the maximum rate,
	// a stable bad rate "ages out" after "age_bad_rate_after_sec".
	AgeBadRateAfterSec int64 `protobuf:"varint,12,opt,name=age_bad_rate_after_sec,json=ageBadRateAfterSec" json:"age_bad_rate_after_sec,omitempty"`
	// bad_rate_increase defines the percentage by which a bad rate will be
	// increased when it's aging out.
	BadRateIncrease float64 `protobuf:"fixed64,13,opt,name=bad_rate_increase,json=badRateIncrease" json:"bad_rate_increase,omitempty"`
	// max_rate_approach_threshold is the fraction of the current rate limit that the actual
	// rate must exceed for the throttler to increase the limit when the replication lag
	// is below target_replication_lag_sec. For example, assuming the actual replication lag
	// is below target_replication_lag_sec, if the current rate limit is 100, then the actual
	// rate must exceed 100*max_rate_approach_threshold for the throttler to increase the current
	// limit.
	MaxRateApproachThreshold float64 `protobuf:"fixed64,14,opt,name=max_rate_approach_threshold,json=maxRateApproachThreshold" json:"max_rate_approach_threshold,omitempty"`
}

func (m *Configuration) Reset()                    { *m = Configuration{} }
func (m *Configuration) String() string            { return proto.CompactTextString(m) }
func (*Configuration) ProtoMessage()               {}
func (*Configuration) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

// GetConfigurationRequest is the payload for the GetConfiguration RPC.
type GetConfigurationRequest struct {
	// throttler_name specifies which throttler to select. If empty, all active
	// throttlers will be selected.
	ThrottlerName string `protobuf:"bytes,1,opt,name=throttler_name,json=throttlerName" json:"throttler_name,omitempty"`
}

func (m *GetConfigurationRequest) Reset()                    { *m = GetConfigurationRequest{} }
func (m *GetConfigurationRequest) String() string            { return proto.CompactTextString(m) }
func (*GetConfigurationRequest) ProtoMessage()               {}
func (*GetConfigurationRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

// GetConfigurationResponse is returned by the GetConfiguration RPC.
type GetConfigurationResponse struct {
	// max_rates returns the configurations for each throttler.
	// It's keyed by the throttler name.
	Configurations map[string]*Configuration `protobuf:"bytes,1,rep,name=configurations" json:"configurations,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *GetConfigurationResponse) Reset()                    { *m = GetConfigurationResponse{} }
func (m *GetConfigurationResponse) String() string            { return proto.CompactTextString(m) }
func (*GetConfigurationResponse) ProtoMessage()               {}
func (*GetConfigurationResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *GetConfigurationResponse) GetConfigurations() map[string]*Configuration {
	if m != nil {
		return m.Configurations
	}
	return nil
}

// UpdateConfigurationRequest is the payload for the UpdateConfiguration RPC.
type UpdateConfigurationRequest struct {
	// throttler_name specifies which throttler to update. If empty, all active
	// throttlers will be updated.
	ThrottlerName string `protobuf:"bytes,1,opt,name=throttler_name,json=throttlerName" json:"throttler_name,omitempty"`
	// configuration is the new (partial) configuration.
	Configuration *Configuration `protobuf:"bytes,2,opt,name=configuration" json:"configuration,omitempty"`
	// copy_zero_values specifies whether fields with zero values should be copied
	// as well.
	CopyZeroValues bool `protobuf:"varint,3,opt,name=copy_zero_values,json=copyZeroValues" json:"copy_zero_values,omitempty"`
}

func (m *UpdateConfigurationRequest) Reset()                    { *m = UpdateConfigurationRequest{} }
func (m *UpdateConfigurationRequest) String() string            { return proto.CompactTextString(m) }
func (*UpdateConfigurationRequest) ProtoMessage()               {}
func (*UpdateConfigurationRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

func (m *UpdateConfigurationRequest) GetConfiguration() *Configuration {
	if m != nil {
		return m.Configuration
	}
	return nil
}

// UpdateConfigurationResponse is returned by the UpdateConfiguration RPC.
type UpdateConfigurationResponse struct {
	// names is the list of throttler names which were updated.
	Names []string `protobuf:"bytes,1,rep,name=names" json:"names,omitempty"`
}

func (m *UpdateConfigurationResponse) Reset()                    { *m = UpdateConfigurationResponse{} }
func (m *UpdateConfigurationResponse) String() string            { return proto.CompactTextString(m) }
func (*UpdateConfigurationResponse) ProtoMessage()               {}
func (*UpdateConfigurationResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{8} }

// ResetConfigurationRequest is the payload for the ResetConfiguration RPC.
type ResetConfigurationRequest struct {
	// throttler_name specifies which throttler to reset. If empty, all active
	// throttlers will be reset.
	ThrottlerName string `protobuf:"bytes,1,opt,name=throttler_name,json=throttlerName" json:"throttler_name,omitempty"`
}

func (m *ResetConfigurationRequest) Reset()                    { *m = ResetConfigurationRequest{} }
func (m *ResetConfigurationRequest) String() string            { return proto.CompactTextString(m) }
func (*ResetConfigurationRequest) ProtoMessage()               {}
func (*ResetConfigurationRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{9} }

// ResetConfigurationResponse is returned by the ResetConfiguration RPC.
type ResetConfigurationResponse struct {
	// names is the list of throttler names which were updated.
	Names []string `protobuf:"bytes,1,rep,name=names" json:"names,omitempty"`
}

func (m *ResetConfigurationResponse) Reset()                    { *m = ResetConfigurationResponse{} }
func (m *ResetConfigurationResponse) String() string            { return proto.CompactTextString(m) }
func (*ResetConfigurationResponse) ProtoMessage()               {}
func (*ResetConfigurationResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{10} }

func init() {
	proto.RegisterType((*MaxRatesRequest)(nil), "throttlerdata.MaxRatesRequest")
	proto.RegisterType((*MaxRatesResponse)(nil), "throttlerdata.MaxRatesResponse")
	proto.RegisterType((*SetMaxRateRequest)(nil), "throttlerdata.SetMaxRateRequest")
	proto.RegisterType((*SetMaxRateResponse)(nil), "throttlerdata.SetMaxRateResponse")
	proto.RegisterType((*Configuration)(nil), "throttlerdata.Configuration")
	proto.RegisterType((*GetConfigurationRequest)(nil), "throttlerdata.GetConfigurationRequest")
	proto.RegisterType((*GetConfigurationResponse)(nil), "throttlerdata.GetConfigurationResponse")
	proto.RegisterType((*UpdateConfigurationRequest)(nil), "throttlerdata.UpdateConfigurationRequest")
	proto.RegisterType((*UpdateConfigurationResponse)(nil), "throttlerdata.UpdateConfigurationResponse")
	proto.RegisterType((*ResetConfigurationRequest)(nil), "throttlerdata.ResetConfigurationRequest")
	proto.RegisterType((*ResetConfigurationResponse)(nil), "throttlerdata.ResetConfigurationResponse")
}

func init() { proto.RegisterFile("throttlerdata.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 711 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x55, 0xdd, 0x4e, 0xdb, 0x4a,
	0x10, 0x96, 0x09, 0xe1, 0xc0, 0x84, 0x00, 0x59, 0x38, 0x60, 0xc2, 0xd1, 0x51, 0x8e, 0xa5, 0xa3,
	0x46, 0x48, 0xcd, 0x45, 0x50, 0x55, 0x5a, 0x54, 0x09, 0x52, 0xaa, 0xaa, 0x55, 0xcb, 0x85, 0x69,
	0x7b, 0xd1, 0x9b, 0xd5, 0xc6, 0x1e, 0x1c, 0x0b, 0xdb, 0xeb, 0xee, 0x2e, 0x25, 0xe9, 0x43, 0xf4,
	0x41, 0x7a, 0xd7, 0x37, 0xea, 0xa3, 0x54, 0xde, 0xdd, 0xfc, 0x12, 0xa0, 0x12, 0x77, 0xde, 0x99,
	0x6f, 0xbe, 0xfd, 0xc6, 0x9e, 0xf9, 0x0c, 0x9b, 0xaa, 0x27, 0xb8, 0x52, 0x09, 0x8a, 0x90, 0x29,
	0xd6, 0xca, 0x05, 0x57, 0x9c, 0x54, 0xa7, 0x82, 0x5e, 0x0d, 0xd6, 0xdf, 0xb3, 0xbe, 0xcf, 0x14,
	0x4a, 0x1f, 0xbf, 0x5c, 0xa1, 0x54, 0xde, 0x77, 0x07, 0x36, 0xc6, 0x31, 0x99, 0xf3, 0x4c, 0x22,
	0x39, 0x86, 0xb2, 0x28, 0x02, 0xae, 0xd3, 0x28, 0x35, 0x2b, 0xed, 0xfd, 0xd6, 0x34, 0xf7, 0x2c,
	0xbe, 0xa5, 0x4f, 0xaf, 0x32, 0x25, 0x06, 0xbe, 0x29, 0xac, 0x1f, 0x02, 0x8c, 0x83, 0x64, 0x03,
	0x4a, 0x97, 0x38, 0x70, 0x9d, 0x86, 0xd3, 0x5c, 0xf1, 0x8b, 0x47, 0xb2, 0x05, 0xe5, 0xaf, 0x2c,
	0xb9, 0x42, 0x77, 0xa1, 0xe1, 0x34, 0x4b, 0xbe, 0x39, 0x3c, 0x5f, 0x38, 0x74, 0xbc, 0x47, 0x50,
	0x3b, 0x47, 0x65, 0xaf, 0xb0, 0x2a, 0x09, 0x81, 0xc5, 0x82, 0x57, 0x33, 0x94, 0x7c, 0xfd, 0xec,
	0xed, 0x03, 0x99, 0x04, 0x5a, 0xe9, 0x5b, 0x50, 0xce, 0x58, 0x6a, 0xa5, 0xaf, 0xf8, 0xe6, 0xe0,
	0xfd, 0x58, 0x82, 0xea, 0x4b, 0x9e, 0x5d, 0xc4, 0xd1, 0x95, 0x60, 0x2a, 0xe6, 0x19, 0x39, 0x82,
	0xba, 0x62, 0x22, 0x42, 0x45, 0x05, 0xe6, 0x49, 0x1c, 0xe8, 0x28, 0x4d, 0x58, 0x44, 0x25, 0x06,
	0xf6, 0x9e, 0x1d, 0x83, 0xf0, 0xc7, 0x80, 0x77, 0x2c, 0x3a, 0xc7, 0x80, 0x3c, 0x81, 0x9d, 0x94,
	0xf5, 0xe7, 0x56, 0x9a, 0x7e, 0xb6, 0x52, 0xd6, 0xbf, 0x59, 0xf6, 0x1f, 0xac, 0xc6, 0x59, 0xac,
	0x62, 0x96, 0x50, 0xdd, 0x4d, 0x49, 0x63, 0x2b, 0x36, 0x56, 0xb4, 0x51, 0x40, 0x0a, 0xe6, 0x38,
	0x0b, 0x04, 0x32, 0x89, 0xee, 0x62, 0xc3, 0x69, 0x3a, 0x7e, 0x25, 0x65, 0xfd, 0x37, 0x36, 0x44,
	0x1e, 0x03, 0xc1, 0x14, 0x45, 0x84, 0x59, 0x30, 0xa0, 0x21, 0x5a, 0x60, 0x59, 0x03, 0x6b, 0xa3,
	0xcc, 0xa9, 0x4d, 0x90, 0xb7, 0xe0, 0xa5, 0x71, 0x46, 0x43, 0xdb, 0x38, 0xed, 0xa2, 0xba, 0x46,
	0xcc, 0x46, 0x57, 0x48, 0x2d, 0x7b, 0x49, 0x4b, 0xf9, 0x37, 0x8d, 0xb3, 0x53, 0x0b, 0xec, 0x18,
	0xdc, 0xf0, 0x5a, 0x59, 0x34, 0x50, 0x70, 0xb1, 0xfe, 0x7d, 0x5c, 0x7f, 0x59, 0x2e, 0xd6, 0xbf,
	0x8f, 0x6b, 0x9e, 0xae, 0x61, 0x47, 0x86, 0x6b, 0xf9, 0x36, 0x5d, 0xc3, 0xfe, 0x34, 0xd7, 0x33,
	0xd8, 0x95, 0xb9, 0x40, 0x16, 0xd2, 0x2e, 0x0b, 0x2e, 0x13, 0x1e, 0x51, 0x16, 0x08, 0x2e, 0x0d,
	0xc5, 0x8a, 0xa6, 0xd8, 0x36, 0x80, 0x8e, 0xc9, 0x9f, 0xe8, 0xb4, 0x2d, 0x8d, 0xa3, 0x8c, 0x0b,
	0xa4, 0x19, 0x95, 0x09, 0xbf, 0x46, 0x39, 0x9a, 0x08, 0xe9, 0x42, 0xc3, 0x69, 0x96, 0xfd, 0x6d,
	0x03, 0x38, 0x3b, 0x37, 0x69, 0xfb, 0x5d, 0x25, 0x79, 0x0a, 0xee, 0xcd, 0xd2, 0x90, 0x67, 0xc9,
	0x40, 0xba, 0x15, 0x5d, 0xf9, 0xf7, 0x4c, 0xa5, 0x49, 0x92, 0x36, 0x6c, 0xb3, 0x08, 0x69, 0x97,
	0x85, 0x7a, 0x0e, 0x28, 0xbb, 0x50, 0x28, 0xb4, 0xd6, 0x55, 0xad, 0x95, 0xb0, 0x08, 0x3b, 0x2c,
	0x2c, 0x06, 0xe2, 0xa4, 0x48, 0x15, 0x3a, 0xf7, 0xa1, 0x36, 0xc2, 0x8f, 0xa6, 0xa3, 0xaa, 0x3f,
	0xfa, 0x7a, 0xd7, 0x60, 0x47, 0x13, 0xf2, 0x02, 0xf6, 0xf4, 0x78, 0x6a, 0xee, 0x3c, 0x17, 0x9c,
	0x05, 0x3d, 0xaa, 0x7a, 0x02, 0x65, 0x8f, 0x27, 0xa1, 0xbb, 0xa6, 0xab, 0xdc, 0xd4, 0x6c, 0xce,
	0x89, 0x05, 0x7c, 0x18, 0xe6, 0xbd, 0x63, 0xd8, 0x79, 0x8d, 0x6a, 0x6a, 0x5d, 0x86, 0x7b, 0xf8,
	0x3f, 0xac, 0x8d, 0xac, 0x80, 0x16, 0xab, 0x65, 0x77, 0x7a, 0xec, 0x33, 0x67, 0x2c, 0x45, 0xef,
	0x97, 0x03, 0xee, 0x4d, 0x0a, 0xbb, 0xa1, 0x01, 0xac, 0x05, 0x93, 0x89, 0xa1, 0xcb, 0x1c, 0xcd,
	0xb8, 0xcc, 0x6d, 0x04, 0xad, 0xa9, 0xa8, 0xb5, 0x9d, 0x19, 0xca, 0x3a, 0x85, 0xcd, 0x39, 0xb0,
	0x39, 0x46, 0xd4, 0x9e, 0x34, 0xa2, 0x4a, 0xfb, 0x9f, 0x19, 0x11, 0xd3, 0x0a, 0x26, 0x6c, 0xea,
	0xa7, 0x03, 0xf5, 0x8f, 0x79, 0xc8, 0x14, 0x3e, 0xe0, 0x45, 0x91, 0x0e, 0x54, 0xa7, 0x84, 0xff,
	0x91, 0x8a, 0xe9, 0x12, 0xd2, 0x84, 0x8d, 0x80, 0xe7, 0x03, 0xfa, 0x0d, 0x05, 0xa7, 0x5a, 0xa0,
	0xd4, 0xce, 0xb2, 0x5c, 0xbc, 0x94, 0x7c, 0xf0, 0x19, 0x05, 0xff, 0xa4, 0xa3, 0xde, 0x01, 0xec,
	0xcd, 0x95, 0x7c, 0xa7, 0x75, 0x76, 0x60, 0xd7, 0x47, 0xf9, 0xb0, 0x79, 0x68, 0x43, 0x7d, 0x1e,
	0xc7, 0x5d, 0xf7, 0x76, 0x97, 0xf4, 0x1f, 0xec, 0xe0, 0x77, 0x00, 0x00, 0x00, 0xff, 0xff, 0xad,
	0x29, 0x6c, 0x04, 0xd8, 0x06, 0x00, 0x00,
}
