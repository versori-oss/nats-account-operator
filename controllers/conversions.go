package controllers

import (
	"github.com/nats-io/jwt"
	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
)

func convertToNATSOperatorLimits(limits v1alpha1.AccountLimits) jwt.OperatorLimits {
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

func convertTimeRanges(times []v1alpha1.StartEndTime) []jwt.TimeRange {
	result := make([]jwt.TimeRange, len(times))
	for _, t := range times {
		result = append(result, jwt.TimeRange{
			Start: t.Start,
			End:   t.End,
		})
	}
	return result
}

func convertToNATSLimits(limits v1alpha1.UserLimits) jwt.Limits {
	return jwt.Limits{
		Max:     int64(limits.Max),
		Payload: int64(limits.Payload),
		Src:     limits.Src,
		Times:   convertTimeRanges(limits.Times),
	}
}

func convertToNATSExportType(ieType v1alpha1.ImportExportType) jwt.ExportType {
	switch ieType {
	case v1alpha1.ImportExportTypeStream:
		return jwt.Stream
	case v1alpha1.ImportExportTypeService:
		return jwt.Service
	default:
		return jwt.Unknown
	}
}

func convertToNATSServiceLatency(latency v1alpha1.AccountServiceLatency) *jwt.ServiceLatency {
	return &jwt.ServiceLatency{
		Sampling: latency.Sampling,
		Results:  jwt.Subject(latency.Results),
	}
}

func convertToNATSImports(imports []v1alpha1.AccountImport) jwt.Imports {
	var result jwt.Imports
	tmp := make([]*jwt.Import, len(imports))
	for _, i := range imports {
		tmp = append(tmp, &jwt.Import{
			Name:    i.Name,
			Subject: jwt.Subject(i.Subject),
			Account: i.Account,
			Token:   i.Token,
			To:      jwt.Subject(i.To),
			Type:    convertToNATSExportType(i.Type),
		})

		result.Add(tmp...)
	}
	return result
}

func convertToNATSExports(exports []v1alpha1.AccountExport) jwt.Exports {
	var result jwt.Exports
	tmp := make([]*jwt.Export, len(exports))
	for _, e := range exports {
		tmp = append(tmp, &jwt.Export{
			Name:                 e.Name,
			Subject:              jwt.Subject(e.Subject),
			Type:                 convertToNATSExportType(e.Type),
			TokenReq:             e.TokenReq,
			ResponseType:         jwt.ResponseType(e.ResponseType),
			Latency:              convertToNATSServiceLatency(*e.ServiceLatency),
			AccountTokenPosition: e.AccountTokenPosition,
		})

		result.Add(tmp...)
	}
	return result
}

func convertToNATSIdentities(idents []v1alpha1.Identity) []jwt.Identity {
	result := make([]jwt.Identity, len(idents))
	for _, i := range idents {
		result = append(result, jwt.Identity{
			ID:    i.ID,
			Proof: i.Proof,
		})
	}
	return result
}
