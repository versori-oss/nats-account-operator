package nsc

import (
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
)

var DefaultOperatorLimits = jwt.OperatorLimits{
	NatsLimits: jwt.NatsLimits{
		Subs:    jwt.NoLimit,
		Data:    jwt.NoLimit,
		Payload: jwt.NoLimit,
	},
	AccountLimits: jwt.AccountLimits{
		Imports:         jwt.NoLimit,
		Exports:         jwt.NoLimit,
		WildcardExports: true,
		DisallowBearer:  false,
		Conn:            jwt.NoLimit,
		LeafNodeConn:    jwt.NoLimit,
	},
	JetStreamLimits: jwt.JetStreamLimits{
		MemoryStorage:        jwt.NoLimit,
		DiskStorage:          jwt.NoLimit,
		Streams:              jwt.NoLimit,
		Consumer:             jwt.NoLimit,
		MaxAckPending:        jwt.NoLimit,
		MemoryMaxStreamBytes: jwt.NoLimit,
		DiskMaxStreamBytes:   jwt.NoLimit,
		MaxBytesRequired:     false,
	},
	JetStreamTieredLimits: map[string]jwt.JetStreamLimits{
		"": {
			MemoryStorage:        jwt.NoLimit,
			DiskStorage:          jwt.NoLimit,
			Streams:              jwt.NoLimit,
			Consumer:             jwt.NoLimit,
			MaxAckPending:        jwt.NoLimit,
			MemoryMaxStreamBytes: jwt.NoLimit,
			DiskMaxStreamBytes:   jwt.NoLimit,
			MaxBytesRequired:     false,
		},
	},
}

func ConvertToNATSOperatorLimits(limits *v1alpha1.AccountLimits) jwt.OperatorLimits {
	l := DefaultOperatorLimits

	if limits == nil {
		return l
	}

	if limits.Subs != l.Subs {
		l.Subs = limits.Subs
	}

	if limits.Conn != l.Conn {
		l.Conn = limits.Conn
	}

	if limits.Leaf != l.LeafNodeConn {
		l.LeafNodeConn = limits.Leaf
	}

	if limits.Imports != l.Imports {
		l.Imports = limits.Imports
	}

	if limits.Exports != l.Exports {
		l.Exports = limits.Exports
	}

	if limits.Data != l.Data {
		l.Data = limits.Data
	}

	if limits.Payload != l.Payload {
		l.Payload = limits.Payload
	}

	if limits.Wildcards != l.WildcardExports {
		l.WildcardExports = limits.Wildcards
	}

	return l
}

func ConvertTimeRanges(times []v1alpha1.StartEndTime) []jwt.TimeRange {
	if times == nil {
		return nil
	}
	result := make([]jwt.TimeRange, len(times))
	for _, t := range times {
		result = append(result, jwt.TimeRange{
			Start: t.Start,
			End:   t.End,
		})
	}
	return result
}

func ConvertToNATSSigningKeys(keys []string) jwt.SigningKeys {
	var result jwt.SigningKeys
	result.Add(keys...)
	return result
}

func ConvertToNATSLimits(limits v1alpha1.UserLimits) jwt.Limits {
	natslimits := jwt.NatsLimits{
		Subs:    0,
		Data:    limits.Max,
		Payload: limits.Payload,
	}

	usrLimits := jwt.UserLimits{
		Src:    jwt.CIDRList(limits.Src),
		Times:  ConvertTimeRanges(limits.Times),
		Locale: "",
	}

	return jwt.Limits{
		UserLimits: usrLimits,
		NatsLimits: natslimits,
	}
}

func ConvertToNATSExportType(ieType v1alpha1.ImportExportType) jwt.ExportType {
	switch ieType {
	case v1alpha1.ImportExportTypeStream:
		return jwt.Stream
	case v1alpha1.ImportExportTypeService:
		return jwt.Service
	default:
		return jwt.Unknown
	}
}

func ConvertToNATSServiceLatency(latency *v1alpha1.AccountServiceLatency) *jwt.ServiceLatency {
	// this field is optional so could potentially be nil
	if latency == nil {
		return nil
	}
	return &jwt.ServiceLatency{
		Sampling: jwt.SamplingRate(latency.Sampling),
		Results:  jwt.Subject(latency.Results),
	}
}

func ConvertToNATSImports(imports []v1alpha1.AccountImport) jwt.Imports {
	var result jwt.Imports
	tmp := make([]*jwt.Import, len(imports))

	for n, i := range imports {
		tmp[n] = &jwt.Import{
			Name:    i.Name,
			Subject: jwt.Subject(i.Subject),
			Account: i.Account,
			Token:   i.Token,
			To:      jwt.Subject(i.To),
			Type:    ConvertToNATSExportType(i.Type),
		}
	}

	result.Add(tmp...)
	return result
}

func ConvertToNATSExports(exports []v1alpha1.AccountExport) jwt.Exports {
	var result jwt.Exports
	tmp := make([]*jwt.Export, len(exports))

	for n, export := range exports {
		tmp[n] = &jwt.Export{
			Name:                 export.Name,
			Subject:              jwt.Subject(export.Subject),
			Type:                 ConvertToNATSExportType(export.Type),
			TokenReq:             export.TokenReq,
			ResponseType:         ConvertToNATSResponseType(export.ResponseType),
			Latency:              ConvertToNATSServiceLatency(export.ServiceLatency),
			AccountTokenPosition: export.AccountTokenPosition,
		}
	}

	result.Add(tmp...)
	return result
}

func ConvertToNATSResponseType(responseType v1alpha1.ResponseType) jwt.ResponseType {
	switch responseType {
	case v1alpha1.ResponseTypeSingleton:
		return jwt.ResponseTypeSingleton
	case v1alpha1.ResponseTypeStream:
		return jwt.ResponseTypeStream
	case v1alpha1.ResponseTypeChunked:
		return jwt.ResponseTypeChunked
	default:
		return ""
	}
}

func ConvertToNATSUserPermissions(permissions v1alpha1.UserPermissions) jwt.Permissions {
	perms := jwt.Permissions{
		Pub: jwt.Permission{
			Allow: permissions.Pub.Allow,
			Deny:  permissions.Pub.Deny,
		},
		Sub: jwt.Permission{
			Allow: permissions.Sub.Allow,
			Deny:  permissions.Sub.Deny,
		},
	}

	if permissions.Resp != nil {
		perms.Resp = &jwt.ResponsePermission{
			MaxMsgs: permissions.Resp.Max,
			Expires: time.Duration(permissions.Resp.TTL) * time.Second,
		}
	}

	return perms
}
