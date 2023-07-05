package nsc

import (
	"github.com/nats-io/jwt/v2"
	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
)

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
	if exports == nil {
		return nil
	}

	result := make(jwt.Exports, len(exports))

	for n, export := range exports {
		result[n] = &jwt.Export{
			Name:                 export.Name,
			Subject:              jwt.Subject(export.Subject),
			Type:                 ConvertToNATSExportType(export.Type),
			TokenReq:             export.TokenReq,
			ResponseType:         ConvertToNATSResponseType(export.ResponseType),
			Latency:              ConvertToNATSServiceLatency(export.ServiceLatency),
			AccountTokenPosition: export.AccountTokenPosition,
		}
	}

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

func ConvertToNatsLimits(in v1alpha1.NatsLimits, defaults jwt.NatsLimits) jwt.NatsLimits {
    return jwt.NatsLimits{
        Subs:    getDefaultFromPtr(in.Subs, defaults.Subs),
        Data:    getDefaultFromPtr(in.Data, defaults.Data),
        Payload: getDefaultFromPtr(in.Payload, defaults.Payload),
    }
}

func ConvertToAccountLimits(in v1alpha1.AccountLimits, defaults jwt.AccountLimits) jwt.AccountLimits {
    return jwt.AccountLimits{
        Imports:         getDefaultFromPtr(in.Imports, defaults.Imports),
        Exports:         getDefaultFromPtr(in.Exports, defaults.Exports),
        WildcardExports: getDefaultFromPtr(in.WildcardExports, defaults.WildcardExports),
        DisallowBearer:  in.DisallowBearer,
        Conn:            getDefaultFromPtr(in.Conn, defaults.Conn),
        LeafNodeConn:    getDefaultFromPtr(in.LeafNodeConn, defaults.LeafNodeConn),
    }
}

func ConvertToNatsTimeRanges(in []v1alpha1.StartEndTime) []jwt.TimeRange {
    if in == nil {
        return nil
    }

    out := make([]jwt.TimeRange, len(in))
    for i, v := range in {
        out[i] = jwt.TimeRange{
            Start: v.Start,
            End:   v.End,
        }
    }

    return out
}
