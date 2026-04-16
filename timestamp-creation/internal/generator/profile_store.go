package generator

import "timestamp-creation/internal/model"

// ProfileStore keeps immutable grouping stats from pass 1.
type ProfileStore struct {
	groups map[model.ProfileKey]model.GroupProfile
}

func NewProfileStore(groups map[model.ProfileKey]model.GroupProfile) ProfileStore {
	if groups == nil {
		groups = make(map[model.ProfileKey]model.GroupProfile)
	}
	return ProfileStore{groups: groups}
}

func (s ProfileStore) Get(key model.ProfileKey) model.GroupProfile {
	if p, ok := s.groups[key]; ok {
		return p
	}
	return model.GroupProfile{}
}
