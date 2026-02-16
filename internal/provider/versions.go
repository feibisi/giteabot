package provider

import (
    "fmt"
    "strconv"
    "strings"
)

func ParseVersionTitle(title string, milestone Milestone) (Version, bool) {
    parts := strings.Split(title, ".")
    if len(parts) < 2 {
        return Version{}, false
    }
    major, err := strconv.Atoi(parts[0])
    if err != nil {
        return Version{}, false
    }
    minor, err := strconv.Atoi(parts[1])
    if err != nil {
        return Version{}, false
    }
    patch := 0
    if len(parts) > 2 {
        patch, err = strconv.Atoi(parts[2])
        if err != nil {
            return Version{}, false
        }
    }
    return Version{
        MajorMinor: fmt.Sprintf("%d.%d", major, minor),
        Milestone:  milestone,
        Patch:      patch,
        Major:      major,
        Minor:      minor,
    }, true
}

func CompareMajorMinor(a, b Version) int {
    if a.Major != b.Major {
        if a.Major > b.Major {
            return 1
        }
        return -1
    }
    if a.Minor != b.Minor {
        if a.Minor > b.Minor {
            return 1
        }
        return -1
    }
    return 0
}

func ComparePatch(a, b Version) int {
    if c := CompareMajorMinor(a, b); c != 0 {
        return c
    }
    if a.Patch > b.Patch {
        return 1
    }
    if a.Patch < b.Patch {
        return -1
    }
    return 0
}
