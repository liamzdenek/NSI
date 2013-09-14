package main

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func main() {
	http.HandleFunc("/", Upload)

	http.ListenAndServe(":8080", nil)
}

func Upload(res http.ResponseWriter, req *http.Request) {
	var captions_str string
	var html string
	var caption_parse_error error

	if req.Method == "POST" {
		req.ParseForm()
		captions_str = req.PostFormValue("captions")
		if len(captions_str) > 0 {
			config := &Config{
				OptGlobalRemoveNewlines:   req.PostFormValue("opt_global_remove_newlines") != "",
				OptGlobalOutputToCSV:      req.PostFormValue("opt_global_output_to_csv") != "",
				OptGlobalOutputToExcelCSV: req.PostFormValue("opt_global_output_to_excel_csv") != "",
				OptGlobalAlphabetize:      req.PostFormValue("opt_global_alphabetize") != "",

				OptSdsIncludeMusicNotes: req.PostFormValue("opt_sds_include_music_notes") != "",
				OptSdsExcludeLowercase:  req.PostFormValue("opt_sds_exclude_lowercase") != "",
				OptSdsExcludeNumeric:    req.PostFormValue("opt_sds_exclude_numeric") != "",
			}
			captions, err := NewCaptions(captions_str, config)
			if err != nil {
				caption_parse_error = err
			} else {
				sds := captions.FindSoundDescriptions(config)
				sids := captions.FindSpeakerIDs()

				if config.OptGlobalAlphabetize {
					sort.Sort(captions)
					sort.Sort(sds)
					sort.Sort(sids)
				}

				html = html + "<h2>Sound Descriptions</h2>" + CaptionDump(sds, config)
				html = html + "<h2>Speaker IDs</h2>" + CaptionDump(sids, config)
				html = html + "<h2>Unfiltered List</h2>" + CaptionDump(captions, config)
			}
		}
	}

	html_head := "<p>Paste a caption file...</p><form action='/' method='POST'>" +
		"<textarea name='captions' cols='80' rows='24'>" + captions_str + "</textarea><br/>"
	if caption_parse_error != nil {
		html_head = html_head +
			"<p style='color:red;'>" + caption_parse_error.Error() + "</p>"
	}
	html_head = html_head +
		"<h3>Global Options</h3>" +
		Checkbox(req, "opt_global_remove_newlines") + " Remove line breaks from the caption text (Useful when importing into Excel)<br/>" +
		Checkbox(req, "opt_global_output_to_csv") + " Output to CSV<br/>" +
		" - " + Checkbox(req, "opt_global_output_to_excel_csv") + " Excel Compatible CSV?<br/>" +
		Checkbox(req, "opt_global_alphabetize") + " Alphabetize? <br/>" +
		"<h3>Sound Description Options</h3>" +
		Checkbox(req, "opt_sds_include_music_notes") + " Include captions that contain musical notes<br/>" +
		Checkbox(req, "opt_sds_exclude_lowercase") + " Exclude captions that are not entirely uppercase<br/>" +
		Checkbox(req, "opt_sds_exclude_numeric") + " Exclude captions that only contain letters<br/>" +
		"<input type='submit'>" +
		"</form>"

	html = html_head + html
	res.Header().Add("Content-Type", "text/html")
	res.Write([]byte(html))
}

func Checkbox(req *http.Request, name string) string {
	if req.PostFormValue(name) != "" {
		return "<input type='checkbox' checked='checked' name='" + name + "'>"
	}
	return "<input type='checkbox' name='" + name + "'>"
}

func CaptionDump(in *Captions, config *Config) string {
	if config.OptGlobalOutputToCSV {
		return CaptionDumpToCSV(in, config)
	} else {
		return CaptionDumpToHTMLTable(in, config)
	}
}

func CaptionDumpToCSV(in *Captions, config *Config) string {
	var s string
	if in.SubsetOf != nil {
		if config.OptGlobalOutputToExcelCSV {
			s = "\"CaptionNumber\",\"Number\",\"StartTime\",\"EndTime\",\"Text\",\"Notes\"\n"
		} else {
			s = "CaptionNumber,Number,StartTime,EndTime,Text,Notes\n"
		}
	} else {
		if config.OptGlobalOutputToExcelCSV {
			s = "\"Number\",\"StartTime\",\"EndTime\",\"Text\",\"Notes\""
		} else {
			s = "Number,StartTime,Endtime,Text,Notes"
		}
	}
	for index, caption := range in.Captions {
		if config.OptGlobalOutputToExcelCSV {
			s = s + "\"" + caption.CaptionNumber + "\","
			if in.SubsetOf != nil {
				s = s + "\"" + strconv.Itoa(index+1) + "\","
			}
			s = s + "\"" + caption.StartTime + "\"," +
				"\"" + caption.EndTime + "\"," +
				"\"" + caption.Text + "\"," +
				"\"" + strings.Join(caption.Notes, "\n") + "\"\n"
		} else {
			s = s + caption.CaptionNumber + ","
			if in.SubsetOf != nil {
				s = s + strconv.Itoa(index) + ","
			}
			s = s + caption.StartTime + "," +
				caption.EndTime + "," +
				strings.Replace(strings.Replace(caption.Text, "\n", "\\n", -1), ",", "\\,", -1) + "," +
				strings.Join(caption.Notes, "\\n") + "\n"
		}
	}
	return "<textarea rows=\"24\" cols=\"80\" readonly>" + s + "</textarea>"
}

func CaptionDumpToHTMLTable(in *Captions, config *Config) string {
	var s string
	s = "<table border=1>\n\t<tr>\n"
	if in.SubsetOf != nil {
		s = s + "<th>CaptionNumber</th><th>Number</th><th>StartTime</th><th>Endtime</th><th>Text</th><th>Notes</th>"
	} else {
		s = s + "<th>Number</th><th>StartTime</th><th>Endtime</th><th>Text</th><th>Notes</th>"
	}
	s = s + "</tr>"
	for index, caption := range in.Captions {
		s = s + "<tr>"
		if in.SubsetOf != nil {
			s = s + "<td><a href='#unfilt_" + caption.CaptionNumber + "'>" + caption.CaptionNumber + "</a></td>"
			s = s + "<td>" + strconv.Itoa(index+1) + "</td>"
		} else {
			s = s + "<td id='unfilt_" + caption.CaptionNumber + "'>" + caption.CaptionNumber + "</td>"
		}
		s = s + "<td>" + caption.StartTime + "</td>"
		s = s + "<td>" + caption.EndTime + "</td>"
		s = s + "<td>" + strings.Replace(caption.Text, "\n", "<br>", -1) + "</td>"
		s = s + "<td>" + strings.Join(caption.Notes, "<br/>") + "</td>"
		s = s + "</tr>"
	}
	s = s + "</table>"
	return s
}
