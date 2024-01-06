package udptypes

import "errors"

func (sched *Scheduler) GetPeerIPFromName(name string) (string, error) {

	for key, value := range sched.PeerDatabase {
		if value.Name == name {
			return key, nil
		}
	}

	return "", errors.New("no peer with that name")
}
