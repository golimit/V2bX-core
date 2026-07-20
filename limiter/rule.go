package limiter

import (
	"regexp"

	"github.com/InazumaV/V2bX/api/panel"
)

func (l *Limiter) CheckDomainRule(destination string) (reject bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for i := range l.DomainRules {
		if l.DomainRules[i].MatchString(destination) {
			reject = true
			break
		}
	}
	return
}

func (l *Limiter) CheckProtocolRule(protocol string) (reject bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for i := range l.ProtocolRules {
		if l.ProtocolRules[i] == protocol {
			reject = true
			break
		}
	}
	return
}

func (l *Limiter) UpdateRule(rule *panel.Rules) error {
	domainRules := make([]*regexp.Regexp, len(rule.Regexp))
	for i := range rule.Regexp {
		domainRules[i] = regexp.MustCompile(rule.Regexp[i])
	}
	l.mu.Lock()
	l.DomainRules = domainRules
	l.ProtocolRules = rule.Protocol
	l.mu.Unlock()
	return nil
}
