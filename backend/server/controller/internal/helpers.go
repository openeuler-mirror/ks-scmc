package internal

import (
	"math"
	"sort"

	pb "scmc/rpc/pb/container"
	"scmc/rpc/pb/security"
)

func resourceLimitDiff(r0, r1 *pb.ResourceLimit) bool {
	if r0 == nil && r1 == nil {
		return false
	} else if r0 != nil && r1 != nil {
		return math.Abs(r0.CpuLimit-r1.CpuLimit) > 1e-9 || r0.CpuPrio != r1.CpuPrio ||
			math.Abs(r0.MemoryLimit-r1.MemoryLimit) > 1e-9 ||
			math.Abs(r0.MemorySoftLimit-r1.MemorySoftLimit) > 1e-9

	}
	return true
}

func restartPolicyDiff(r0, r1 *pb.RestartPolicy) bool {
	if r0 == nil && r1 == nil {
		return false
	} else if r0 != nil && r1 != nil {
		return r0.Name != r1.Name || r0.MaxRetry != r1.MaxRetry
	}
	return true
}

func networkDiff(n0, n1 []*pb.NetworkConfig) bool {
	if len(n0) != len(n1) {
		return true
	}
	sort.Slice(n0, func(p, q int) bool {
		return n0[p].Interface < n0[q].Interface
	})
	sort.Slice(n1, func(p, q int) bool {
		return n1[p].Interface < n1[q].Interface
	})
	for i := 0; i < len(n0); i++ {
		p0, p1 := n0[i], n1[i]
		if p0 == nil && p1 == nil {
			continue
		} else if p0 != nil && p1 != nil {
			if p0.Interface != p1.Interface || p0.IpAddress != p1.IpAddress || p0.MacAddress != p1.MacAddress {
				return true
			}
		}
	}
	return false
}

func containerBasicConfigDiff(cfgs *pb.ContainerConfigs, in *pb.UpdateRequest) bool {
	// log.Debugf("containerBasicConfigDiff\n\t%+v\n\t%+v", cfgs, in)

	if cfgs == nil && in == nil {
		return false
	} else if cfgs != nil && in != nil {
		return resourceLimitDiff(cfgs.ResouceLimit, in.ResourceLimit) ||
			restartPolicyDiff(cfgs.RestartPolicy, in.RestartPolicy) ||
			networkDiff(cfgs.Networks, in.Networks)
	}
	return true
}

func isEmptySecurityConfig(c *pb.SecurityConfig) bool {
	if c == nil {
		return true
	}

	proc := c.ProcProtection == nil || (!c.ProcProtection.IsOn && len(c.ProcProtection.ExeList) == 0)
	nproc := c.NprocProtection == nil || (!c.NprocProtection.IsOn && len(c.NprocProtection.ExeList) == 0)
	file := c.FileProtection == nil || (!c.FileProtection.IsOn && len(c.FileProtection.FileList) == 0)
	net := c.NetworkRule == nil || (!c.NetworkRule.IsOn)
	cmd := !c.DisableCmdOperation

	return proc && nproc && file && net && cmd
}

func procProtectionDiff(p0, p1 *security.ProcProtection) bool {
	if p0 == nil && p1 == nil {
		return false
	} else if p0 != nil && p1 != nil {
		if p0.IsOn != p1.IsOn {
			return true
		} else if strSliceDiff(p0.ExeList, p1.ExeList) {
			return true
		}
	}
	return false
}

func fileProtectionDiff(f0, f1 *security.FileProtection) bool {
	if f0 == nil && f1 == nil {
		return false
	} else if f0 != nil && f1 != nil {
		if f0.IsOn != f1.IsOn {
			return true
		} else if strSliceDiff(f0.FileList, f1.FileList) {
			return true
		}
	}
	return false
}

func strSliceDiff(s0, s1 []string) bool {
	if len(s0) != len(s1) {
		return true
	}

	for i := 0; i < len(s0); i++ {
		if s0[i] != s1[i] {
			return true
		}
	}

	return false
}

func networkRuleDiff(n0, n1 *security.NetworkRule) bool {
	if n0 == nil && n1 == nil {
		return false
	} else if n0 != nil && n1 != nil {
		return strSliceDiff(n0.Protocols, n1.Protocols) || n0.Addr != n1.Addr || n0.Port != n1.Port
	}
	return true
}

func networkRulesDiff(n0, n1 *security.NetworkRuleList) bool {
	if n0 == nil && n1 == nil {
		return false
	} else if n0 != nil && n1 != nil {
		if n0.IsOn != n1.IsOn {
			return true
		} else if len(n0.Rules) != len(n1.Rules) {
			return true
		}
		for i := 0; i < len(n0.Rules); i++ {
			if networkRuleDiff(n0.Rules[i], n1.Rules[i]) {
				return true
			}
		}
	}
	return false
}

func containerSecurityConfigDiff(s0, s1 *pb.SecurityConfig) bool {
	// log.Debugf("containerSecurityConfigDiff\n\t%+v\n\t%+v", s0, s1)

	if s0 == nil && s1 == nil {
		return false
	} else if s0 != nil && s1 != nil {
		return procProtectionDiff(s0.ProcProtection, s1.ProcProtection) ||
			procProtectionDiff(s0.NprocProtection, s1.NprocProtection) ||
			fileProtectionDiff(s0.FileProtection, s1.FileProtection) ||
			networkRulesDiff(s0.NetworkRule, s1.NetworkRule) ||
			s0.DisableCmdOperation != s1.DisableCmdOperation
	}
	return true
}
