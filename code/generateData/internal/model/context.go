package model

import "sync"

type GenerationContext struct {
    mu sync.Mutex

    RecentComplaints []Complaint

    EntityFrequency map[string]int

    ActiveCampaigns []Campaign
}

func NewContext() *GenerationContext {
    return &GenerationContext{
        EntityFrequency: make(map[string]int),
    }
}