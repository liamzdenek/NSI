package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

var ILLEGAL_FILENAME_CHARS []string = []string{"%", "!", "/", ".", "\\", "_", "'", "\"", "?", "=", "+", "&"}
var WORKDIR = "files/"

func main() {
	http.HandleFunc("/upload", Upload)
	http.HandleFunc("/report", Report)
	http.HandleFunc("/delete", Delete)
	http.HandleFunc("/search", Search)
	http.HandleFunc("/", Index)

	http.ListenAndServe(":8080", nil)
}

func Index(res http.ResponseWriter, req *http.Request) {
	var html string

	html = "<a href='/upload'>Add a file</a><br/>"

	html = html + "<a href='/search'>Search</a>"
	files, err := ioutil.ReadDir(WORKDIR)
	if err != nil {
		html = html + "<p>There was an error getting the file listing: " + err.Error() + "</p>"
	}

	html = html + "<ul>"
	for _, fileinfo := range files {
		if fileinfo.IsDir() == false {
			html = html + "<li><a href='/report?f=" + fileinfo.Name() + "'>" + fileinfo.Name() + "</a> - <a href='/delete?f=" + fileinfo.Name() + "'>Delete</a></li>"
		}
	}
	html = html + "</ul>"

	res.Header().Add("Content-Type", "text/html")
	res.Write([]byte(html))
}

func Search(res http.ResponseWriter, req *http.Request) {
	var html string
	{
		files, err := ioutil.ReadDir(WORKDIR)
		if err != nil {
			html = html + "<p>There was an error getting the file listing: " + err.Error() + "</p>"
		}

		html = "<script>function check_all(val){es = document.getElementsByTagName('input');for(i in es){e = es[i];if(e.name && e.name.substr(0,5)=='file_'){e.checked=val;}}}</script>"
		html = html + "<a href='/'>Back to Index</a><br/>"
		html = html + "<form method='POST'><input name='query' value='" + req.FormValue("query") + "'><br/>Search in the following files: <input type='button' onclick='check_all(true)' value='Check All'> <input type='button' onclick='check_all(false)' value='Uncheck All'></br>"
		for _, fileinfo := range files {
			if fileinfo.IsDir() == false {
				html = html + "-" + Checkbox(req, "file_"+fileinfo.Name()) + fileinfo.Name() + "<br/>"
			}
		}

		html_opts, config := GetInputs(req, false)
		html = html + html_opts
		html = html + "<input type='submit'></form>"

		config.IsSearch = true

		if req.Method == "POST" {
			query := strings.ToLower(req.FormValue("query"))
			if len(query) > 0 {
				var captions *Captions = &Captions{}
				for _, fileinfo := range files {
					if fileinfo.IsDir() == false {
						if len(req.Form["file_"+fileinfo.Name()]) > 0 && len(req.Form["file_"+fileinfo.Name()][0]) != 0 {
							if -1 != strings.IndexAny(fileinfo.Name(), strings.Join(ILLEGAL_FILENAME_CHARS, "")) {
								res.WriteHeader(400)
								res.Write([]byte("Filename contains an illegal character"))
								return
							}
							f, err := ioutil.ReadFile(WORKDIR + "/" + fileinfo.Name())
							if err != nil {
								res.WriteHeader(403)
								res.Write([]byte("Internal error reading " + fileinfo.Name() + ": " + err.Error()))
								return
							}
							var captions_ *Captions
							captions_, err = NewCaptions(string(f), config)
							captions_.Name = fileinfo.Name()
							if err != nil {
								res.WriteHeader(403)
								res.Write([]byte("Internal error parsing captions for " + fileinfo.Name() + ": " + err.Error()))
								return
							}
							captions = CaptionsMerge(captions, captions_.Reduce(query))
						}
					}
				}
				fmt.Printf("Got captions: %v\n", captions)
				html_, err := ShowReportByCaptions(captions, config)
				if err != nil {
					html = html + "Error generating report: " + err.Error() + "<br/>"
				} else {
					html = html + html_
				}
			}
		}
	}
	res.Header().Add("Content-Type", "text/html")
	res.Write([]byte(html))
}

func Delete(res http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	filename := req.FormValue("f")
	if len(filename) == 0 {
		res.WriteHeader(404)
		res.Write([]byte("File name not provided"))
		return
	}
	if -1 != strings.IndexAny(filename, strings.Join(ILLEGAL_FILENAME_CHARS, "")) {
		res.WriteHeader(400)
		res.Write([]byte("Filename contains an illegal character"))
		return
	}
	_, err := ioutil.ReadFile(WORKDIR + "/" + filename)
	if err != nil {
		res.WriteHeader(403)
		res.Write([]byte("File couldn't be read; does it exist?: " + err.Error()))
		return
	}

	if req.FormValue("now") == "true" {
		err := os.Remove(WORKDIR + "/" + filename)
		if err != nil {
			res.WriteHeader(403)
			res.Write([]byte("File couldn't be deleted: " + err.Error()))
			return
		}
		res.Header().Add("Location", "/")
		res.WriteHeader(302)
		res.Write([]byte{})
		return
	}

	res.Header().Add("Content-Type", "text/html")
	res.Write([]byte("You are about to delete <strong>" + filename + "</strong>. Are you sure? This cannot be undone.<br/><br/><a href='?f=" + filename + "&now=true'>Delete " + filename + " Irrevocably</a>"))
	//res.Header().Add("Content-Type", "text/html")
	//res.Write([]byte("This hasn't been implemented yet. Press the back button."))
}

func Report(res http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	var file string
	{
		filename := req.FormValue("f")
		if len(filename) == 0 {
			res.WriteHeader(404)
			res.Write([]byte("File name not provided"))
			return
		}
		if -1 != strings.IndexAny(filename, strings.Join(ILLEGAL_FILENAME_CHARS, "")) {
			res.WriteHeader(400)
			res.Write([]byte("Filename contains an illegal character"))
			return
		}
		file_, err := ioutil.ReadFile(WORKDIR + "/" + filename)
		if err != nil {
			res.WriteHeader(403)
			res.Write([]byte("File couldn't be read: " + err.Error()))
			return
		}
		file = string(file_)
	}
	html_opts, config := GetInputs(req, false)

	// don't need to handle the validation since input is guaranteed to already be valid
	html, _ := ShowReport(file, config)

	html = "<a href='/'>Back to Index</a><form method='POST'>" + html_opts + "<input type='submit'></form>" + html

	res.Header().Add("Content-Type", "text/html")
	res.Write([]byte(html))
}

func Upload(res http.ResponseWriter, req *http.Request) {
	var captions_str string
	var html, html_opts string
	var caption_parse_error error
	var errorlist []error
	var post_location string = ""

	if req.Method == "POST" {
		req.ParseForm()
		var config *Config
		html_opts, config = GetInputs(req, true)
		captions_str = req.PostFormValue("captions")
		if len(captions_str) > 0 {

			html, caption_parse_error = ShowReport(captions_str, config)
			if caption_parse_error == nil && config.OptSave {
				if len(config.OptSaveFilename) == 0 {
					errorlist = append(errorlist, errors.New("A file name was not specified"))
				} else if -1 != strings.IndexAny(config.OptSaveFilename, strings.Join(ILLEGAL_FILENAME_CHARS, "")) {
					errorlist = append(errorlist, errors.New("The filename contains an illegal character"))
				} else {
					err := ioutil.WriteFile(WORKDIR+"/"+config.OptSaveFilename, []byte(captions_str), 0666)
					if err != nil {
						errorlist = append(errorlist, err)
					}
					post_location = "/report?f=" + config.OptSaveFilename
					html_opts, _ = GetInputs(req, false)
				}
			}
		}
	} else {
		html_opts, _ = GetInputs(req, true)
	}

	var html_head string
	html_head = "<a href='/'>Back to Index</a><br/>"
	if len(post_location) > 0 {
		html_head = html_head + "<form method='POST' action='" + post_location + "'>"
	} else {
		html_head = html_head + "<form method='POST'>"
	}
	if caption_parse_error != nil {
		html_head = html_head +
			"<p style='color:red;'>" + caption_parse_error.Error() + "</p>"
	}
	html_head = html_head + html_opts +
		"<input type='submit'>" +
		"</form>"

	html = html_head + html
	res.Header().Add("Content-Type", "text/html")
	res.Write([]byte(html))
}

func GetInputs(req *http.Request, is_upload bool) (html_opts string, config *Config) {
	if is_upload {
		html_opts = html_opts +
			"<p>Paste a caption file...</p>" +
			"<textarea name='captions' cols='80' rows='24'>" + req.PostFormValue("captions") + "</textarea><br/>" +
			"<h3>Save Options</h3>" +
			Checkbox(req, "opt_save") + "Save this caption file for later use?<br/>" +
			"- <input name='opt_save_filename'> File name. may not contain any of the following: " + strings.Join(ILLEGAL_FILENAME_CHARS, " ") + "<br/>"
	}
	html_opts = html_opts +
		"<h3>Global Options</h3>" +
		Checkbox(req, "opt_global_remove_newlines") + " Remove line breaks from the caption text (Useful when importing into Excel)<br/>" +
		Checkbox(req, "opt_global_output_to_csv") + " Output to CSV<br/>" +
		" - " + Checkbox(req, "opt_global_output_to_excel_csv") + " Excel Compatible CSV?<br/>" +
		Checkbox(req, "opt_global_alphabetize") + " Alphabetize? <br/>" +
		"<h3>Non-Speech Options</h3>" +
		Checkbox(req, "opt_sds_include_music_notes") + " Include captions that contain musical notes (any of the following: " + strings.Join(NOTECHARS, ", ") + ")<br/>" +
		Checkbox(req, "opt_sds_exclude_lowercase") + " Exclude captions that are not entirely uppercase<br/>" +
		Checkbox(req, "opt_sds_exclude_numeric") + " Exclude captions that do not contain any letters<br/>"
	config = &Config{
		OptGlobalRemoveNewlines:   req.PostFormValue("opt_global_remove_newlines") != "",
		OptGlobalOutputToCSV:      req.PostFormValue("opt_global_output_to_csv") != "",
		OptGlobalOutputToExcelCSV: req.PostFormValue("opt_global_output_to_excel_csv") != "",
		OptGlobalAlphabetize:      req.PostFormValue("opt_global_alphabetize") != "",

		OptSave:         req.PostFormValue("opt_save") != "",
		OptSaveFilename: req.PostFormValue("opt_save_filename"),

		OptSdsIncludeMusicNotes: req.PostFormValue("opt_sds_include_music_notes") != "",
		OptSdsExcludeLowercase:  req.PostFormValue("opt_sds_exclude_lowercase") != "",
		OptSdsExcludeNumeric:    req.PostFormValue("opt_sds_exclude_numeric") != ""}
	return
}

func ShowReport(captions_str string, config *Config) (html string, err error) {
	captions, err := NewCaptions(captions_str, config)
	if err != nil {
		return "", err
	} else {
		return ShowReportByCaptions(captions, config)
	}
}

func ShowReportByCaptions(captions *Captions, config *Config) (html string, err error) {
	sds := captions.FindSoundDescriptions(config)
	sids := captions.FindSpeakerIDs()

	if config.OptGlobalAlphabetize {
		sort.Sort(captions)
		sort.Sort(sds)
		sort.Sort(sids)
	}

	html = html + "<h2>Non-Speech</h2>" + CaptionDump(sds, config)
	html = html + "<h2>Speaker IDs</h2>" + CaptionDump(sids, config)
	html = html + "<h2>Unfiltered List</h2>" + CaptionDump(captions, config)
	return
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
			s = "\"CaptionNumber\",\"Number\""
			if config.IsSearch {
				s = s + "\"Source\","
			}
			s = s + "\"StartTime\",\"EndTime\",\"Text\",\"Notes\"\n"
		} else {
			s = "CaptionNumber,Number,"
			if config.IsSearch {
				s = s + "Source,"
			}
			s = s + "StartTime,EndTime,Text,Notes\n"
		}
	} else {
		if config.OptGlobalOutputToExcelCSV {
			s = "\"Number\",\"Source\",\"StartTime\",\"EndTime\",\"Text\",\"Notes\""
		} else {
			s = "Number,Source,StartTime,Endtime,Text,Notes"
		}
	}
	for index, caption := range in.Captions {
		if config.OptGlobalOutputToExcelCSV {
			s = s + "\"" + caption.CaptionNumber + "\","
			if in.SubsetOf != nil {
				s = s + "\"" + strconv.Itoa(index+1) + "\","
			}
			if config.IsSearch {
				if caption.Parent != nil {
					s = s + "\"" + caption.Parent.Name + "\","
				} else {
					s = s + "\"[[Unavailable]]\""
				}
			}
			s = s + "\"" + caption.StartTime + "\"," +
				"\"" + caption.EndTime + "\"," +
				"\"" + caption.Text + "\"," +
				"\"" + strings.Join(caption.Notes, "\n") + "\"\n"
		} else {
			s = s + caption.CaptionNumber + ","
			if in.SubsetOf != nil {
				s = s + strconv.Itoa(index+1) + ","
			}
			if config.IsSearch {
				if caption.Parent != nil {
					s = s + caption.Parent.Name + ","
				} else {
					s = s + "[[Unavailable]],"
				}
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
		if config.IsSearch {
			s = s + "<th>CaptionNumber</th><th>Number</th><th>Source</th><th>StartTime</th><th>Endtime</th><th>Text</th><th>Notes</th>"
		} else {
			s = s + "<th>CaptionNumber</th><th>Number</th><th>StartTime</th><th>Endtime</th><th>Text</th><th>Notes</th>"
		}
	} else {
		if config.IsSearch {
			s = s + "<th>Number</th><th>Source</th><th>StartTime</th><th>Endtime</th><th>Text</th><th>Notes</th>"

		} else {
			s = s + "<th>Number</th><th>StartTime</th><th>Endtime</th><th>Text</th><th>Notes</th>"

		}
	}
	s = s + "</tr>"
	for index, caption := range in.Captions {
		s = s + "<tr>"
		if in.SubsetOf != nil {
			if config.IsSearch && caption.Parent != nil {
				s = s + "<td><a href='/report?f=" + caption.Parent.Name + "#unfilt_" + caption.CaptionNumber + "'>" + caption.CaptionNumber + "</a></td>"
			} else if config.IsSearch {
				s = s + "<td>" + caption.CaptionNumber + "</td>"
			} else {
				s = s + "<td><a href='#unfilt_" + caption.CaptionNumber + "'>" + caption.CaptionNumber + "</a></td>"
			}
			s = s + "<td>" + strconv.Itoa(index+1) + "</td>"
		} else {
			s = s + "<td id='unfilt_" + caption.CaptionNumber + "'>" + caption.CaptionNumber + "</td>"
		}
		if config.IsSearch {
			if caption.Parent != nil {
				s = s + "<td>" + caption.Parent.Name + "</td>"
			} else {
				s = s + "<td> [[Unavailable]] </td>"
			}
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
