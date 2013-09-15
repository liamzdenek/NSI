package main

import (
	"strings"
	"errors"
	"strconv"
)

var NOTECHARS []string = []string{"&#9833;", // quarter note
				"&#9834;", //eighth note
				"&#9835;", //single bar note
				"&#9836;", //double bar note
				"&#9837;", // flat note
				"&#9838;", // natural note
				"&#9839;"} // sharp note

type Config struct {
	OptGlobalRemoveNewlines bool
	OptGlobalOutputToCSV bool
	OptGlobalOutputToExcelCSV bool
	OptGlobalAlphabetize bool

	OptSdsIncludeMusicNotes bool
	OptSdsExcludeLowercase bool
	OptSdsExcludeNumeric bool
}

type Caption struct {
	StartTime     string
	EndTime       string
	CaptionNumber string
	Text          string
	Notes         []string
}

func NewCaption(s, newl string, config *Config) (*Caption, bool) {
	parts := strings.SplitN(s, newl, -1)
	if len(parts) < 3 {
		return nil, true
	}
	times := strings.SplitN(parts[1], " --> ", 2)
	if config.OptGlobalRemoveNewlines {
		newl = ""
	}
	return &Caption{
		StartTime:     times[0],
		EndTime:       times[1],
		CaptionNumber: parts[0],
		Text:          strings.Replace(strings.Join(parts[2:], newl), "\r\n", "\n", -1),
	}, false
}

type Captions struct {
	Captions []*Caption
	Raw      string
	SubsetOf *Captions
}

func NewCaptions(s string, config *Config) (*Captions, error) {
	captions := make([]*Caption, 0)

	newl := "\r\n"
	s = strings.Trim(s, newl);

	parts := strings.SplitN(s, newl+newl, -1)

	if len(parts) == 0 {
		newl = "\n"
		parts = strings.SplitN(s, newl+newl, -1)
	}

	for index, caption := range parts {
		c, err := NewCaption(caption, newl, config)
		if err != false {
			return nil, errors.New("There was an error parsing caption #"+strconv.Itoa(index+1)+"; this is probably an encoding error or a malformed caption")
		}
		captions = append(captions, c)
	}
	return &Captions{Captions: captions, Raw: s}, nil
}

func (c *Captions) FindSpeakerIDs() *Captions {
	sids := make([]*Caption, 0)
	for _, caption := range c.Captions {
		t := caption.Text
		sid_colon := strings.IndexAny(t, ":")

		if sid_colon != -1 {
			//sid_speaker := t[:sid_colon]
			sids = append(sids, caption)
		}
	}
	return &Captions{Captions: sids, SubsetOf: c, Raw: c.Raw}
}

func (c *Captions) FindSoundDescriptions(config *Config) *Captions {
	sds := make([]*Caption, 0)
	for _, caption := range c.Captions {
		t := caption.Text
		//t = strings.Replace(t, "\r\n", "", -1)
		sd_start := strings.IndexAny(t, "([{")
		sd_end := strings.IndexAny(t, ")]}")

		do_append := false
		notes := make([]string, 0)

		if sd_start != -1 && sd_end != -1 {
			sd := t[sd_start+1 : sd_end]
			if strings.ToUpper(sd) != sd {
				if config.OptSdsExcludeLowercase {
					continue
				}
				notes = append(notes, "Is not entirely uppercase")
			}
			if strings.IndexAny(sd, "abcdefghijklmnopqrstuvwzyzABCDEFGHIJKLMNOPQRSTUVWXYZ") == -1 {
				if config.OptSdsExcludeNumeric {
					continue
				}
				notes = append(notes, "Does not contain any letters")
			}
			do_append = true
		}

		if config.OptSdsIncludeMusicNotes {
						for _, note := range NOTECHARS {
				if strings.Contains(t, note) {
					notes = append(notes, "Sound Description contains a music note")
					do_append = true
					break
				}
			}
		}

		if do_append {
			sds = append(sds, &Caption{
				StartTime:     caption.StartTime,
				EndTime:       caption.EndTime,
				CaptionNumber: caption.CaptionNumber,
				Text:          caption.Text,
				Notes:         notes,
			})
		}
	}
	return &Captions{Captions: sds, SubsetOf: c, Raw: c.Raw}
}

/*
Satisfy sort.Interface for an alphabetical sort
*/
func (c *Captions) Len() int {
	return len(c.Captions)
}

func (c *Captions) Less(i, j int) bool {
	return strings.ToLower(c.Captions[i].Text) < strings.ToLower(c.Captions[j].Text)
}

func (c *Captions) Swap(i, j int) {
	c.Captions[i], c.Captions[j] = c.Captions[j], c.Captions[i]
}

