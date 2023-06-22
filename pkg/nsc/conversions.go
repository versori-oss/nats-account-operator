package nsc

import (
	"time"

	"github.com/nats-io/jwt"
	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
)

func ConvertToNATSOperatorLimits(limits v1alpha1.AccountLimits) jwt.OperatorLimits {
	return jwt.OperatorLimits{
		Subs:            int64(limits.Subs),
		Conn:            int64(limits.Conn),
		LeafNodeConn:    int64(limits.Leaf),
		Imports:         int64(limits.Imports),
		Exports:         int64(limits.Exports),
		Data:            int64(limits.Data),
		Payload:         int64(limits.Payload),
		WildcardExports: limits.Wildcards,
	}
}

func ConvertTimeRanges(times []v1alpha1.StartEndTime) []jwt.TimeRange {
	result := make([]jwt.TimeRange, len(times))
	for _, t := range times {
		result = append(result, jwt.TimeRange{
			Start: t.Start,
			End:   t.End,
		})
	}
	return result
}

func ConvertToNATSLimits(limits v1alpha1.UserLimits) jwt.Limits {
	return jwt.Limits{
		Max:     int64(limits.Max),
		Payload: int64(limits.Payload),
		Src:     limits.Src,
		Times:   ConvertTimeRanges(limits.Times),
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
		Sampling: latency.Sampling,
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

func ConvertToNATSIdentities(idents []v1alpha1.Identity) []jwt.Identity {
	result := make([]jwt.Identity, len(idents))
	for n, ident := range idents {
		result[n] = jwt.Identity{
			ID:    ident.ID,
			Proof: ident.Proof,
		}
	}
	return result
}

func ConvertToNATSUserPermissions(permissions v1alpha1.UserPermissions) jwt.Permissions {
	return jwt.Permissions{
		Pub: jwt.Permission{
			Allow: permissions.Pub.Allow,
			Deny:  permissions.Pub.Deny,
		},
		Sub: jwt.Permission{
			Allow: permissions.Sub.Allow,
			Deny:  permissions.Sub.Deny,
		},
		Resp: &jwt.ResponsePermission{
			MaxMsgs: permissions.Resp.Max,
			Expires: time.Duration(permissions.Resp.TTL) * time.Second,
		},
	}
}
