package bundleinstall

import (
	"github.com/paketo-buildpacks/packit"
)

type PlanEntryResolver struct{}

func NewPlanEntryResolver() PlanEntryResolver {
	return PlanEntryResolver{}
}

func (r PlanEntryResolver) Resolve(entries []packit.BuildpackPlanEntry) packit.BuildpackPlanEntry {
	chosenEntry := entries[0]

	if chosenEntry.Metadata == nil {
		chosenEntry.Metadata = map[string]interface{}{}
	}

	for _, entry := range entries {
		if entry.Metadata["build"] == true {
			chosenEntry.Metadata["build"] = true
		}
		if entry.Metadata["launch"] == true {
			chosenEntry.Metadata["launch"] = true
		}
	}

	return chosenEntry
}
