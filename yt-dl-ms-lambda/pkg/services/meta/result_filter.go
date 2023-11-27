package meta

import (
	spotify "github.com/zmb3/spotify/v2"
)

func Filter_results(in []TrackMeta) []TrackMeta {
	m := map[spotify.ID]TrackMeta{}
	out := []TrackMeta{}
	for _, r := range in {
		m[r.ID] = r
	}
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
