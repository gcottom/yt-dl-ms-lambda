package meta

import (
	"fmt"
	"regexp"
	"strings"
)

func GetArtistTitleCombos(filename, author string) map[string][]string {
	filename = strings.ReplaceAll(filename, ":", "-")
	t, c := pSanitize(filename)
	return artistTitleSplit(t, c, author)
}

func pSanitize(s string) (sanitizedTrack, coverArtist string) {
	inparReg := regexp.MustCompile(`\([^)]*\)`)
	inpar := inparReg.FindAllStringSubmatch(s, -1)
	san := inparReg.ReplaceAllString(s, "")
	for _, match := range inpar {
		if strings.Contains(strings.ToLower(strings.ReplaceAll(match[0], " ", "")), "albumversion") || strings.Contains(strings.ToLower(strings.ReplaceAll(match[0], " ", "")), "officialmusicvideo") || strings.Contains(strings.ToLower(strings.ReplaceAll(match[0], " ", "")), "liveversion") || strings.Contains(strings.ToLower(strings.ReplaceAll(match[0], " ", "")), "officialvideo") || strings.Contains(strings.ToLower(strings.ReplaceAll(match[0], " ", "")), "officiallyricvideo") {
			continue
		}
		if strings.Contains(strings.ToLower(match[0]), "cover by") || strings.Contains(strings.ToLower(match[0]), "by ") {
			match[0] = string([]byte(match[0])[0 : len(match[0])-1])
			fmt.Println("filename contains \"cover by\" or \"by\"")
			t := strings.Split(match[0], "by")
			return san, t[1]

		}
		if strings.Contains(strings.ToLower(match[0]), "cover") {
			match[0] = string([]byte(match[0])[1 : len(match[0])-1])
			fmt.Println("filename contains \"cover\"")
			fmt.Println("P String:", match[0])
			if strings.HasSuffix(strings.ToLower(match[0]), "cover") {
				return san, match[0]

			}
		}
	}
	return san, ""
}
func artistTitleSplit(s, c, a string) map[string][]string {
	m := make(map[string][]string)
	emojiReg := regexp.MustCompile(`(\u00a9|\u00ae|[\u2000-\u3300]|\ud83c[\ud000-\udfff]|\ud83d[\ud000-\udfff]|\ud83e[\ud000-\udfff])`)
	s = strings.ReplaceAll(strings.ReplaceAll(strings.Trim(s, " "), ",", ""), "  ", "")
	s = atStripping(s)
	c = strings.ReplaceAll(strings.ReplaceAll(strings.Trim(c, " "), ",", ""), "  ", "")
	c = atStripping(c)
	a = strings.ReplaceAll(strings.ReplaceAll(strings.Trim(a, " "), ",", ""), "  ", "")
	a = atStripping(a)

	//filter out emojis
	s = emojiReg.ReplaceAllString(s, "")
	c = emojiReg.ReplaceAllString(c, "")
	a = emojiReg.ReplaceAllString(a, "")
	if strings.Contains(s, "-") {
		sp := strings.Split(s, "-")
		//cover artist overrides original artist
		if c != "" && len(sp) == 2 {
			m[strings.Trim(sanitizeAuthor(c), " ")] = []string{strings.Trim(sp[0], " "), strings.Trim(sp[1], " ")}
			return m
		}
		//artist - title case
		if c == "" && len(sp) == 2 {
			if strings.EqualFold(sanitizeAuthor(strings.Trim(a, " ")), strings.Trim(sp[0], " ")) {
				m[sanitizeAuthor(strings.Trim(sp[0], " "))] = []string{strings.Trim(sp[1], " "), strings.Trim(sp[0]+"-"+sp[1], " ")}
			} else {
				m[sanitizeAuthor(strings.Trim(sp[0], " "))] = []string{strings.Trim(sp[1], " ")}
				m[sanitizeAuthor(strings.Trim(a, " "))] = []string{strings.Trim(sp[0]+"-"+sp[1], " ")}
			}
			m[sanitizeAuthor(strings.Trim(sp[1], " "))] = []string{strings.Trim(sp[0], " ")}
			return m
		}
		//artist - title-title case or
		//artist-artist - title case
		if c == "" && len(sp) == 3 {
			if strings.EqualFold(sanitizeAuthor(strings.Trim(a, " ")), strings.Trim(sp[0], " ")) {
				m[sanitizeAuthor(strings.Trim(sp[0], " "))] = []string{strings.Trim(sp[1]+"-"+sp[2], " "), strings.Trim(sp[0]+"-"+sp[1]+"-"+sp[2], " ")}
			} else {
				m[sanitizeAuthor(strings.Trim(sp[0], " "))] = []string{strings.Trim(sp[1]+"-"+sp[2], " ")}
				m[sanitizeAuthor(strings.Trim(a, " "))] = []string{strings.Trim(sp[0]+"-"+sp[1]+"-"+sp[2], " ")}
			}
			m[sanitizeAuthor(strings.Trim(sp[0]+"-"+sp[1], " "))] = []string{strings.Trim(sp[2], " ")}
			m[sanitizeAuthor(strings.Trim(sp[1]+"-"+sp[2], " "))] = []string{strings.Trim(sp[0], " ")}
			m[sanitizeAuthor(strings.Trim(sp[2], " "))] = []string{strings.Trim(sp[0]+"-"+sp[1], " ")}
			return m
		}
		//artist - title-title-title
		//artist-artist - title-title
		//artits-artits-artist - title
		if c == "" && len(sp) == 4 {
			m[sanitizeAuthor(strings.Trim(sp[0], " "))] = []string{strings.Trim(sp[1]+"-"+sp[2]+"-"+sp[3], " ")}
			m[sanitizeAuthor(strings.Trim(sp[0]+"-"+sp[1], " "))] = []string{strings.Trim(sp[2]+"-"+sp[3], " ")}
			m[sanitizeAuthor(strings.Trim(sp[2]+"-"+sp[3], " "))] = []string{strings.Trim(sp[0]+"-"+sp[1], " ")}
			m[sanitizeAuthor(strings.Trim(sp[0]+"-"+sp[1]+"-"+sp[2], " "))] = []string{strings.Trim(sp[3], " ")}
			m[sanitizeAuthor(strings.Trim(sp[1]+"-"+sp[2]+"-"+sp[3], " "))] = []string{strings.Trim(sp[0], " ")}
		}

		return m
	}
	m[sanitizeAuthor(a)] = []string{s}
	if c != "" {
		m[c] = []string{s}
	}

	return m

}
func sanitizeAuthor(a string) string {
	a = strings.ToLower(a)
	a = strings.ReplaceAll(a, " - official", "")
	a = strings.ReplaceAll(a, "-official", "")
	a = strings.ReplaceAll(a, "official", "")
	a = strings.ReplaceAll(a, " - vevo", "")
	a = strings.ReplaceAll(a, "-vevo", "")
	a = strings.ReplaceAll(a, "vevo", "")
	a = strings.ReplaceAll(a, "@", "")
	a = strings.ReplaceAll(a, " - topic", "")
	a = strings.ReplaceAll(a, "-topic", "")
	a = strings.ReplaceAll(a, "topic", "")
	a = strings.Trim(a, " ")
	return a
}

func atStripping(s string) string {
	if strings.Contains(s, "@") {
		sp := strings.Split(s, "@")
		san := ""
		//check if the @ was at the end or beggining of the artist name
		if len(sp) > 1 {
			//exactly 1 @ between the beggining and end of the artist
			if len(sp) == 2 {
				san = sp[0]
				//split by space to elimate the @ed artist
				sp2 := strings.Split(sp[1], " ")
				if len(sp2) > 1 {
					for _, sp2s := range sp2[1:] {
						san = san + sp2s + " "
					}
				}
			}
			if len(sp) > 2 {
				san = sp[0]
				sp2 := strings.Split(sp[1], " ")
				if len(sp2) > 1 {
					for _, sp2s := range sp2[1:] {
						san = san + sp2s + " "
					}
				}
				for _, sp1 := range sp[2:] {
					san = san + "@" + sp1
				}
				san = strings.ReplaceAll(san, "  ", "")
				san = strings.Trim(san, " ")
				san = atStripping(san)
			}
		}
		if san != "" {
			s = san
		}
		return s
	}
	return s
}
