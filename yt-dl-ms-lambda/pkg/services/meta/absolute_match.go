package meta

import (
	"fmt"
	"regexp"
	"strings"
)

func Find_absolute_match(tMeta []TrackMeta, art, tit string) (found bool, result TrackMeta) {
	absolute_match := TrackMeta{}
	absolute_match_found := false
	for _, r := range tMeta {
		//check the entire meta artist string against the passed artist string
		absolute_match_found, absolute_match = check_match(art, tit, r.Artist, r)
		if absolute_match_found {
			return absolute_match_found, absolute_match
		}
		//if the metadata contains mutiple artists, there may be trouble getting an
		//absolute match. split the artists apart and check each one to increase the
		//odds that an absolute match will be found
		sp := strings.Split(r.Artist, ", ")
		//if len(sp) ! >1, there was not multiple artists in the meta and this check is redundant
		//so skip double checking the meta
		if len(sp) > 1 {
			for _, spa := range sp {
				absolute_match_found, absolute_match = check_match(art, tit, spa, r)
				if absolute_match_found {
					return absolute_match_found, absolute_match
				}
			}
		}
	}
	return absolute_match_found, absolute_match
}
func check_match(art, tit, mart string, r TrackMeta) (found bool, result TrackMeta) {
	absolute_match := TrackMeta{}
	//check if there is a match for artist and title combo. perform feat stripping on the
	//title to ensure that false negatives are less likely
	if strings.EqualFold(art, mart) && (strings.EqualFold(tit, r.Title) || strings.EqualFold(strings.Trim(saniParens(featStripping(tit)), " "), strings.Trim(saniParens(featStripping(r.Title)), " "))) {
		absolute_match.Artist = r.Artist
		absolute_match.Album = r.Album
		absolute_match.AlbumArt = r.AlbumArt
		absolute_match.Title = r.Title
		fmt.Println("Absolute Match Found")
		return true, absolute_match
	}
	return false, absolute_match
}
func saniParens(s string) string {
	inparReg := regexp.MustCompile(`\([^)]*\)`)
	san := inparReg.ReplaceAllString(s, "")
	san = strings.ReplaceAll(san, "  ", " ")
	return san
}
