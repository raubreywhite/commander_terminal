package main

import (
	// Standard library packages
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"regexp"

	"github.com/jroimartin/gocui"
	"github.com/jessevdk/go-flags"
	"github.com/jung-kurt/gofpdf"
	"gopkg.in/mgo.v2/bson"
	"github.com/raubreywhite/commander_backend/models"

	"io"
	"io/ioutil"
	"log"
)

var u = models.User{}
var ps = []models.Project{}
var es = []models.Entry{}
var level = "main"
var editing = "nothing"
var client = "Unknown"
var cs = []string{}
var psGivenClient = []int{}
var projectPos = 0
var entryPos = 0

func PadLeft(str string, length int) string {
	for {
		if len(str) >= length {
			return str[0: length]
		}
		str = "0" + str
	}
}

func GetIntsNow()(int,int,int,int,int){
	currentYear , _ := strconv.Atoi(time.Now().Local().Format("2006"))
	currentMonth , _ := strconv.Atoi(time.Now().Local().Format("1"))
	currentDay , _ := strconv.Atoi(time.Now().Local().Format("2"))
	currentHour , _ := strconv.Atoi(time.Now().Local().Format("15"))
	currentMin , _ := strconv.Atoi(time.Now().Local().Format("4"))
	
	return currentYear, currentMonth, currentDay, currentHour, currentMin
}

func ConvertIntsToTime(year int, month int, day int, hour int, min int) time.Time {
	currentTime := strconv.Itoa(year) + "/" +
		strconv.Itoa(month) + "/" +
		strconv.Itoa(day) + "/" +
		strconv.Itoa(hour) + "/" +
		strconv.Itoa(min)
	retval, _ := time.Parse("2006/1/2/15/4", currentTime)
	return retval
}

func ConvertIntsToString(year int, month int, day int, hour int, min int) string {
	retval := strconv.Itoa(year) + "/" +
		PadLeft(strconv.Itoa(month),2) + "/" +
		PadLeft(strconv.Itoa(day),2) + " " +
		PadLeft(strconv.Itoa(hour),2) + ":" +
		PadLeft(strconv.Itoa(min),2)
	return retval
}

func ConvertIntsToStringDate(year int, month int, day int) string {
	retval := strconv.Itoa(year) + "/" +
		PadLeft(strconv.Itoa(month),2) + "/" +
		PadLeft(strconv.Itoa(day),2)
	return retval
}

func ConvertTimeToInts(t time.Time) (int, int, int, int, int) {
	year , _ := strconv.Atoi(t.Format("2006"))
	month , _ := strconv.Atoi(t.Format("1"))
	day , _ := strconv.Atoi(t.Format("2"))
	hour , _ := strconv.Atoi(t.Format("15"))
	min , _ := strconv.Atoi(t.Format("4"))
	
	return year, month, day, hour, min
}

func IncreaseInts(year int, month int, day int, hour int, min int, incMin int)(int,int,int,int,int){
	t := ConvertIntsToTime(year, month, day, hour, min).Add(time.Minute*time.Duration(incMin))
	return ConvertTimeToInts(t)
}


func EntryToRow() []string {

	rate := strconv.Itoa(es[entryPos].Rate)
	if rate=="" {
		rate= " "
	}

	startTime := ConvertIntsToString(es[entryPos].StartYear, es[entryPos].StartMonth, es[entryPos].StartDay, es[entryPos].StartHour, es[entryPos].StartMin)

	endTime := ConvertIntsToString(es[entryPos].EndYear, es[entryPos].EndMonth, es[entryPos].EndDay, es[entryPos].EndHour, es[entryPos].EndMin)

	category := es[entryPos].Category

	subcategory := es[entryPos].Subcategory

	info := es[entryPos].Info

	status := es[entryPos].Status
	if status=="" {
		status = " "
	}

	startDate := ConvertIntsToStringDate(es[entryPos].StartYear, es[entryPos].StartMonth, es[entryPos].StartDay)

	duration := ConvertIntsToTime(es[entryPos].EndYear, es[entryPos].EndMonth, es[entryPos].EndDay, es[entryPos].EndHour, es[entryPos].EndMin).Sub(ConvertIntsToTime(es[entryPos].StartYear, es[entryPos].StartMonth, es[entryPos].StartDay, es[entryPos].StartHour, es[entryPos].StartMin)).Hours()

	money := strconv.FormatFloat(float64(es[entryPos].Rate)*duration,'f', 0, 64)

	return []string{
		rate,
		startTime,
		endTime,
		category,
		subcategory,
		info,
		status,
		money,
		startDate,
		strconv.FormatFloat(duration,'f', 2, 64)}
}

func ProjectToRow() []string {

	client := ps[projectPos].Client
	name := ps[projectPos].Name

	return []string{
		name,
		client}
}

type YearMonth struct {
	Year int
	Month int
	Client string
	Name string
	ProjectPos int
	Processed bool
}

func retainchars(str, chr string) string {
    return strings.Map(func(r rune) rune {
        if strings.IndexRune(chr, r) >= 0 {
            return r
        }
        return -1
    }, str)
}

func genPDF() {

	var yearMonths []YearMonth
	for pIndex, p := range ps {
		projectPos = pIndex
		downloadEntries()

		for _, e := range es {
			eYM := YearMonth {e.StartYear, e.StartMonth, p.Client, p.Name, projectPos, false}
			unique := true
			for _, x := range yearMonths {
				if x == eYM {
					unique = false
				}
			}
			if unique {
				yearMonths = append(yearMonths, eYM)
			}
		}
	}
	for _, ymClientYearMonth := range yearMonths {
	//	ymClientYearMonth :=  yearMonths[0]

		billed := 0
		hours := 0.0

		pdfContainsInfo := false
		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()

		pdf.SetFont("Helvetica", "B", 12)
		_, lineHt := pdf.GetFontSize()
		//currentYear, currentMonth, currentDay, _, _ := GetIntsNow()
		pdf.Write(lineHt, "INVOICE / "+strings.ToUpper(ymClientYearMonth.Client)+" / " +strconv.Itoa(ymClientYearMonth.Year)+"_"+strconv.Itoa(ymClientYearMonth.Month)+"\n\n\n\n")

		for _, ymProjectPos := range yearMonths {
			if ymProjectPos.Processed {
				continue
			}
			if ymProjectPos.Year != ymClientYearMonth.Year {
				continue
			}
			if ymProjectPos.Month != ymClientYearMonth.Month {
				continue
			}
			if ymProjectPos.Client != ymClientYearMonth.Client {
				continue
			}
			ymProjectPos.Processed = true

			projectPos = ymProjectPos.ProjectPos
			downloadEntries()

			pdf.SetFont("Helvetica", "B", 12)
			_, lineHt := pdf.GetFontSize()
			pdf.Write(lineHt, "PROJECT / " + strings.ToUpper(ymProjectPos.Name) + "\n\n")

			pdf.SetFont("Helvetica", "B", 8)
			w := []float64{20.0, 15.0, 15.0, 15.0, 20.0, 25.0, 70.0}
			wSum := 0.0
			for _, v := range w {
				wSum += v
			}
			// 	Header
			header := []string{"DATE", "HOURS", "RATE", "BILLED" ,"CATEGORY", "SUBCATEGORY","DESCRIPTION"}
			for j, str := range header {
				if j==0 {
					pdf.CellFormat(w[j], 7, str, "LB", 0, "C", false, 0, "")
				} else {
					pdf.CellFormat(w[j], 7, str, "B", 0, "C", false, 0, "")
				}
			}
			pdf.Ln(-1)
			// Data
			pdf.SetFont("Helvetica", "", 8)
			for index, e := range es {
				entryPos=index
				if ymProjectPos.Year != e.StartYear {
					continue
				}
				if ymProjectPos.Month != e.StartMonth {
					continue
				}
				x := EntryToRow()
				fmt.Println(x)
				temp, _ := strconv.Atoi(x[7])
				billed = billed + temp
				temp2, _ := strconv.ParseFloat(x[9], 64)
				hours = hours + temp2

				pdf.CellFormat(w[0], 6, x[8], "L", 0, "C", false, 0, "")
				pdf.CellFormat(w[1], 6, x[9], "", 0, "C", false, 0, "")
				pdf.CellFormat(w[2], 6, x[0], "", 0, "C", false, 0, "")
				pdf.CellFormat(w[3], 6, x[7], "", 0, "C", false, 0, "")
				pdf.CellFormat(w[4], 6, x[3], "", 0, "C", false, 0, "")
				pdf.CellFormat(w[5], 6, x[4], "", 0, "C", false, 0, "")
				pdf.CellFormat(w[6], 6, x[5], "", 0, "", false, 0, "")

				pdf.Ln(-1)

				pdfContainsInfo = true
			}
			pdf.CellFormat(wSum, 0, "", "T", 0, "", false, 0, "")

			pdf.Write(lineHt, "\n\n\n\n")
		}

		pdf.SetFont("Helvetica", "", 8)
		_, lineHt = pdf.GetFontSize()
		pdf.Write(lineHt, "HOURS WORKED / " + strconv.FormatFloat(hours,'f', 2, 64) + " \n\n")
		pdf.Write(lineHt, "TOTAL BILLED / NOK " + strconv.Itoa(billed) + "\n\n")

		if pdfContainsInfo {
			client := retainchars(strings.ToLower(ymClientYearMonth.Client),"qwertyuiopasdfghjklzxcvbnm")
			folder := "Documents/billing/"+client
			os.MkdirAll(folder,os.ModeDir)
			file := folder+"/"+strconv.Itoa(ymClientYearMonth.Year)+"_"+strconv.Itoa(ymClientYearMonth.Month)+"_"+client+".pdf"
			fmt.Println(file)
			pdf.OutputFileAndClose(file)
		} else {
			_ = pdf.OutputFileAndClose("/dev/null")
		}
	}
}

func uploadUser() {
	jsonStr, err := json.Marshal(u)
	req, err := http.NewRequest("POST", "http://localhost:8080/edit/users", bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&u)
}

func uploadProject() {
	jsonStr, err := json.Marshal(ps[projectPos])
	req, err := http.NewRequest("POST", "http://localhost:8080/edit/projects", bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&ps[projectPos])
	establishProjectsGivenClient()
}

func uploadEntry() {
	jsonStr, err := json.Marshal(es[entryPos])
	req, err := http.NewRequest("POST", "http://localhost:8080/edit/entries", bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&es[entryPos])
}

func createProject() {
	p := models.Project{Session: u.Session, Name: "A new project", Client: "Unknown"}

	jsonStr, err := json.Marshal(p)
	req, err := http.NewRequest("POST", "http://localhost:8080/create/projects", bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&p)

	ps = append(ps, p)
	u.Projects = append(u.Projects, p.Id)
	uploadUser()

	if !u.LoggedIn {
		fmt.Println("Failed login")
	}
	establishProjectsGivenClient()
}



func createEntry(arg string) {
	//name := "A new entry"

	currentYear, currentMonth, currentDay, currentHour, currentMin := GetIntsNow()
	endYear, endMonth, endDay, endHour, endMin := IncreaseInts(currentYear, currentMonth, currentDay, currentHour, currentMin, 60)

	e := models.Entry{Session: u.Session,
		Status: "",
		Rate: 800,
		StartYear: currentYear,
		StartMonth: currentMonth,
		StartDay : currentDay,
		StartHour: currentHour,
		StartMin: currentMin,
		EndYear: endYear,
		EndMonth: endMonth,
		EndDay : endDay,
		EndHour : endHour,
		EndMin : endMin,
		Category: "Category",
		Subcategory: "Subcategory",
		Info: "A new entry"}

	jsonStr, err := json.Marshal(e)
	req, err := http.NewRequest("POST", "http://localhost:8080/create/entries", bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&e)

	es = append(es, e)
	ps[projectPos].Entries = append(ps[projectPos].Entries, e.Id)
	uploadProject()
}

func getProject(Id bson.ObjectId) models.Project {
	p := models.Project{Session: u.Session, Id: Id}

	jsonStr, err := json.Marshal(p)
	req, err := http.NewRequest("POST", "http://localhost:8080/get/projects", bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&p)

	return (p)
}

func getEntry(Id bson.ObjectId) models.Entry {
	e := models.Entry{Session: u.Session, Id: Id}

	jsonStr, err := json.Marshal(e)
	req, err := http.NewRequest("POST", "http://localhost:8080/get/entries", bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&e)

	return (e)
}

func establishProjectsGivenClient(){
	//c = "Unknown"
	cs = []string{}
	psGivenClient = []int{}

	for i, p := range ps {
		unique := true
		for _, x := range cs {
			if x == p.Client {
				unique = false
			}
		}
		if unique {
			cs = append(cs, p.Client)
		}

		if client == p.Client {
			psGivenClient = append(psGivenClient, i)
		}
	}
}

func downloadProjects() {
	ps = []models.Project{}

	for _, Id := range u.Projects {
		p := getProject(Id)
		if(p.Client==""){
			p.Client = "Unknown"
		}
		ps = append(ps, p)
	}
	establishProjectsGivenClient()
}

func printProjects() {
	if len(ps) == 0 {
		fmt.Println("No projects")
	}
	for index, p := range ps {
		fmt.Print(index, ": ", p.Name, "\n")
	}
}

func downloadEntries() {
	es = []models.Entry{}
	for _, Id := range ps[projectPos].Entries {
		e := getEntry(Id)
		es = append(es, e)
	}
}

func printEntries() {
	if len(es) == 0 {
		fmt.Println("No entries")
	}
	for index, e := range es {
		fmt.Print(index, ": ", e.Info, "\n")
	}
}

func deleteEntry(){
	e := models.Entry{Session: u.Session, Id: es[entryPos].Id}

	jsonStr, err := json.Marshal(e)
	req, err := http.NewRequest("POST", "http://localhost:8080/delete/entries", bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	es = append(es[:entryPos],es[entryPos+1:]...)
	ps[projectPos].Entries = append(ps[projectPos].Entries[:entryPos],ps[projectPos].Entries[entryPos+1:]...)
	uploadProject()

	entryPos = 0
}


func deleteProject(){
	downloadEntries()
	for len(ps[projectPos].Entries) > 0 {
		entryPos = 0
		deleteEntry()
	}
	p := models.Project{Session: u.Session, Id: ps[projectPos].Id}

	jsonStr, err := json.Marshal(p)
	req, err := http.NewRequest("POST", "http://localhost:8080/delete/projects", bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	ps = append(ps[:projectPos],ps[projectPos+1:]...)
	u.Projects = append(u.Projects[:projectPos],u.Projects[projectPos+1:]...)
	uploadUser()

	entryPos = 0
}


func projects() bool {
	exit := false
	var input string
	fmt.Println("projects_name")
	if len(u.Projects) > 0 {
		fmt.Println(len(u.Projects), "projects:")

		ps = nil
		for index, Id := range u.Projects {
			p := getProject(Id)
			ps = append(ps, p)
			fmt.Print(index, ": ", p.Name, "\n")
		}
	}

	for !exit {
		if len(u.Projects) == 0 {
			fmt.Println("No projects")
		}
		fmt.Print("p> ")
		fmt.Scan(&input)
		switch {
		case input == "c":
			createProject()
		case input == "g":
			genPDF()
		case input == "m":
			exit = true
		case input == "q":
			return true
		default:
			fmt.Println("Invalid request. 'm' for main, 'q' for exit")
		}
	}
	return false
}

func loginInternal(email string, password string) {
	u = models.User{Email: email, Password: password}

	jsonStr, err := json.Marshal(u)
	req, err := http.NewRequest("POST", "http://localhost:8080/login", bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&u)

	if !u.LoggedIn {
		fmt.Println("Failed login")
	} else {
		downloadProjects()
	}
}

func login() {
	var email string
	var password string
	fmt.Print("Email: ")
	fmt.Scan(&email)
	fmt.Print("Password: ")
	fmt.Scan(&password)

	loginInternal(email, password)
}

func parseFlags(s []string) (string, string) {
	if len(s) == 0 {
		return "", ""
	} else if len(s) == 1 {
		return s[0], ""
	}
	command := s[0]
	args := ""
	for i, val := range s {
		if i == 0 {
			continue
		} else if i == 1 {
			args = val
		} else {
			args = args + " " + val
		}
	}
	return command, args
}

func ls(args string) {
	switch {
	case level == "main":
		printProjects()
	case level == "project":
		printEntries()
	}
}

func cd(args string) {
	switch {
	case args == "":
	  level = "main"
	case args == "..":
		switch {
		case level == "project":
			level = "main"
		}
	default:
		temp, err := strconv.ParseInt(args, 10, 10)
		if err == nil {
			n := int(temp)
			if n >= 0 && n < len(ps) {
				level = "project"
				projectPos = n
				downloadEntries()
			} else {
				fmt.Println("error")
			}
		} else {
			fmt.Println(err)
		}
	}
}

func touch(args string) {
	switch {
	case level == "main":
		createProject()
	case level == "project":
		createEntry("")
	}
}

func rm(args string) {
	switch {
	case level == "project" && args=="project":
		deleteProject()
		level = "main"
	case level == "project" && args=="":
		fmt.Println("'rm -f project' to delete this project")
	case level == "project":
		temp, err := strconv.ParseInt(args, 10, 10)
		if err == nil {
			n := int(temp)
			if n >= 0 && n < len(es) {
				entryPos = n
				deleteEntry()
			} else {
				fmt.Println("error")
			}
		}
	}
}

func mainx() {

	exit := false
	scanner := bufio.NewScanner(os.Stdin)
	var input string

	loginInternal("r@rwhite.no", "hello")

	for !exit {
		switch {
		case level == "main":
			fmt.Print(u.Email, "> ")
		case level == "project":
			fmt.Print("Project ", projectPos, "> ")
		default:
			fmt.Print("> ")
		}
		scanner.Scan()
		input = scanner.Text()

		var opts struct {
			List   bool   `short:"l" long:"list" description:"list"`
			Create bool   `short:"c" long:"create" description:"create"`
			Client string `long:"client" description:"client"`
			Force  bool   `short:"f" long:"force" description:"force"`
			Name   string `long:"name" description:"name"`
		}

		args := strings.Split(input, " ")
		args, _ = flags.ParseArgs(&opts, args)
		command, argsx := parseFlags(args)

		switch {
		case u.LoggedIn:
			switch {
			case command == "cd":
				cd(argsx)
			case command == "ls":
				ls(argsx)
			case command == "rm":
				if !opts.Force {
					fmt.Println("Needs -f flag to rm")
				} else {
					rm(argsx)
				}
			case command == "touch":
				touch(argsx)
			case command == "q":
				exit = true
			}
		case command == "l":
			login()
		case command == "q":
			exit = true
		default:
			fmt.Println("Invalid request. 'l' for login, 'q' for exit")
		}
	}

	fmt.Println("Goodbye")
}

func printProjectsx(v *gocui.View) {
	v.Clear()
	if len(ps) == 0 {
		fmt.Fprintln(v,"No projects available. Create a new one?")
	} else {
		for index, p := range ps {
			fmt.Fprint(v,index, ": ", p.Name, "\n")
		}
		fmt.Fprintln(v,"Create new project")
	}
}

func printEntriesx(v *gocui.View) error {
	v.Clear()
	if len(es) == 0 {
		fmt.Fprintln(v,"No entries available. Create a new one?")
	} else {
		for index, _ := range es {
			entryPos=index
			x := EntryToRow()
			fmt.Fprintf(v,"%v | %v | %v\n",x[0],x[1],x[2])
			}

		//	fmt.Fprint(v,index, ": ", e.Info, "\n")
		//}
		fmt.Fprintln(v,"Create new entry")
	}
	return nil
}


func selectProject(g *gocui.Gui, v *gocui.View) error {
	_, cy := v.Cursor()
	if(cy<len(ps)){
		projectPos=psGivenClient[cy]
		downloadEntries()
		return displayEntries(g,v)
	}
	return nil
}

func selectEntry(g *gocui.Gui, v *gocui.View) error {
	_, cy := v.Cursor()
	if(cy<len(es)){
		return nil
	} else if(cy>=len(es)){
		createEntry("")
		printEntriesx(v)
	}
	return nil
}

func guiNewProject(g *gocui.Gui, v *gocui.View) error {
	createProject()
	printProjectsx(v)
	return nil
}

func guiNewEntry(g *gocui.Gui, v *gocui.View) error {
	createEntry("")
	refreshEntries(g,v)
	return nil
}




func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
	}
	if v.Name()=="clients"{
		_, cy := v.Cursor()
		vb, _ := v.Line(cy)
		client = vb
		establishProjectsGivenClient()
		refreshProjects(g,v)
	}
	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
	if v.Name()=="clients"{
		_, cy := v.Cursor()
		vb, _ := v.Line(cy)
		client = vb
		establishProjectsGivenClient()
		refreshProjects(g,v)
	}
	return nil
}

func printEntries_val(v *gocui.View, pos int) error {
	v.Clear()
	if len(es) == 0 {

	} else {
		for index, _ := range es {
			entryPos=index
			x := EntryToRow()
			fmt.Fprintf(v,"%v\n",x[pos])
			}
	}
	return nil
}

func printProjects_val(v *gocui.View, pos int) error {
	v.Clear()
	if len(ps) == 0 {

	} else {
		for _, index := range psGivenClient {
			projectPos=index
			x := ProjectToRow()
			fmt.Fprintf(v,"%v\n",x[pos])
			}
	}
	return nil
}

func printClients(v *gocui.View) error {
	v.Clear()
	if len(cs) == 0 {

	} else {
		for _, c := range cs {
			fmt.Fprintf(v,"%v\n",c)
		}
		establishProjectsGivenClient()
	}
	return nil
}


func guiNextEntryView(g *gocui.Gui, v *gocui.View) error {
	var err error
	cx, cy := v.Cursor()
	v.Highlight = false
	switch {
		case v.Name()=="entries_rate":
			v, err = g.SetCurrentView("entries_start")
		case v.Name()=="entries_start":
			v, err = g.SetCurrentView("entries_end")
		case v.Name()=="entries_end":
			v, err = g.SetCurrentView("entries_category")
		case v.Name()=="entries_category":
			v, err = g.SetCurrentView("entries_subcategory")
		case v.Name()=="entries_subcategory":
			v, err = g.SetCurrentView("entries_info")
		case v.Name()=="entries_info":
			v, err = g.SetCurrentView("entries_status")
		case v.Name()=="entries_status":
			v, err = g.SetCurrentView("entries_rate")
		case v.Name()=="clients":
			v, err = g.SetCurrentView("projects_name")
		case v.Name()=="projects_name":
			v, err = g.SetCurrentView("projects_client")
		case v.Name()=="projects_client":
			v, err = g.SetCurrentView("clients")
	}
	v.Highlight = true
	v.SetCursor(cx, cy)

	if v.Name()=="clients"{
		_, cy := v.Cursor()
		vb, _ := v.Line(cy)
		client = vb
		establishProjectsGivenClient()
		refreshProjects(g,v)
	}
	return err
}

func guiPreviousEntryView(g *gocui.Gui, v *gocui.View) error {
	var err error
	cx, cy := v.Cursor()
	v.Highlight = false
	switch {
		case v.Name()=="entries_rate":
			v, err = g.SetCurrentView("entries_status")
		case v.Name()=="entries_start":
			v, err = g.SetCurrentView("entries_rate")
		case v.Name()=="entries_end":
			v, err = g.SetCurrentView("entries_start")
		case v.Name()=="entries_category":
			v, err = g.SetCurrentView("entries_end")
		case v.Name()=="entries_subcategory":
			v, err = g.SetCurrentView("entries_category")
		case v.Name()=="entries_info":
			v, err = g.SetCurrentView("entries_subcategory")
		case v.Name()=="entries_status":
			v, err = g.SetCurrentView("entries_info")
		case v.Name()=="clients":
			v, err = g.SetCurrentView("projects_client")
		case v.Name()=="projects_name":
			v, err = g.SetCurrentView("clients")
		case v.Name()=="projects_client":
			v, err = g.SetCurrentView("projects_name")
	}
	v.Highlight = true
	v.SetCursor(cx, cy)

	if v.Name()=="clients"{
		_, cy := v.Cursor()
		vb, _ := v.Line(cy)
		client = vb
		establishProjectsGivenClient()
		refreshProjects(g,v)
	}
	return err
}

func refreshProjects(g *gocui.Gui, v *gocui.View) error {
	var err error
	original := v.Name()
	establishProjectsGivenClient()

	v, err = g.SetCurrentView("clients")
	printClients(v)

	for i, n := range [...]string{"projects_name", "projects_client"}   {
		v, err = g.SetCurrentView(n)
		printProjects_val(v,i)
	}
	v, err = g.SetCurrentView(original)
	return err

}

func refreshEntries(g *gocui.Gui, v *gocui.View) error {
	var err error
	original := v.Name()
	for i, n := range [...]string{"entries_rate", "entries_start", "entries_end", "entries_category", "entries_subcategory","entries_info","entries_status","entries_money"}   {
		v, err = g.SetCurrentView(n)
		printEntries_val(v,i)
	}
	v, err = g.SetCurrentView(original)
	return err
}

func displayEntries(g *gocui.Gui, v *gocui.View) error {
	var err error
	maxX, maxY := g.Size()
	if v, err = g.SetView("entries_rate", -1, -1, 5, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		printEntries_val(v,0)
		//v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
	}

	if v, err = g.SetView("entries_start", 5, -1, 22, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		printEntries_val(v,1)
		//v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
	}

	if v, err = g.SetView("entries_end", 22, -1, 39, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		printEntries_val(v,2)
		//v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
	}

	if v, err = g.SetView("entries_category", 39, -1, 55, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		printEntries_val(v,3)
		//v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
	}

	if v, err = g.SetView("entries_subcategory", 55, -1, 71, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		printEntries_val(v,4)
		//v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
	}

	if v, err = g.SetView("entries_info", 71, -1, maxX-10, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		printEntries_val(v,5)
		//v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
	}

	if v, err = g.SetView("entries_status", maxX-10, -1, maxX-8, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		printEntries_val(v,6)
		//v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
	}

	if v, err = g.SetView("entries_money", maxX-8, -1, maxX, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		printEntries_val(v,7)
		//v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
	}


	if v, err = g.SetCurrentView("entries_info"); err != nil {
		return err
	}
	v.Highlight = true
	return nil
}

func delMsg(g *gocui.Gui, v *gocui.View) error {
	if err := g.DeleteView("entries"); err != nil {
		return err
	}
	if _, err := g.SetCurrentView("projects_name"); err != nil {
		return err
	}
	return nil
}

func verifyEditable(v *gocui.View) bool {
	_, cy := v.Cursor()

	switch {
		case v.Name()=="projects_name":
			if(cy<len(ps)){
				projectPos=psGivenClient[cy]
				return true
			} else {
				return false
			}
		default:
			if(cy<len(es)){
				entryPos=cy
				return true
			} else {
				return false
			}
	}

	return true

}

func guiEdit(g *gocui.Gui, v *gocui.View) error {
	if !verifyEditable(v) {
		return nil
	}

	editing = v.Name()


	if(editing=="entries_status"){
		switch {
		case es[entryPos].Status=="":
			es[entryPos].Status="B"
		case es[entryPos].Status=="B":
			es[entryPos].Status="+"
		case es[entryPos].Status=="+":
			es[entryPos].Status=""
		}
		//es[entryPos].Status=""
		uploadEntry()
		refreshEntries(g,v)
		return nil
	}

	maxX, maxY := g.Size()
	if v, err := g.SetView("editing", maxX/2-30, maxY/2, maxX/2+30, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true

		if _, err := g.SetCurrentView("editing"); err != nil {
			return err
		}

		switch {
		case editing=="projects_name":
			fmt.Fprint(v,ps[projectPos].Name)
		case editing=="entries_rate":
			fmt.Fprint(v,EntryToRow()[0])
		case editing=="entries_start":
			fmt.Fprint(v,EntryToRow()[1])
		case editing=="entries_end":
			fmt.Fprint(v,EntryToRow()[2])
		case editing=="entries_category":
			fmt.Fprint(v,EntryToRow()[3])
		case editing=="entries_subcategory":
			fmt.Fprint(v,EntryToRow()[4])
		case editing=="entries_info":
			fmt.Fprint(v,EntryToRow()[5])
		}

	}
	return nil
}

func guiEditConfirm(g *gocui.Gui, v *gocui.View) error {
	var err error
	_, cy := v.Cursor()
	vb, _ := v.Line(cy)

	switch {
		case editing=="projects_name":
			ps[projectPos].Name = vb
			uploadProject()
		case editing=="projects_client":
			ps[projectPos].Client = vb
			uploadProject()
		case editing=="entries_rate":
			es[entryPos].Rate, _ = strconv.Atoi(vb)
			uploadEntry()
		case editing=="entries_start":
			r, _ := regexp.Compile("[ ]*([0-9]*)[ ]*/[ ]*([0-9]*)[ ]*/[ ]*([0-9]*)[ ]*([0-9]*)[ ]*:[ ]*([0-9]*)[ ]*")
			x := r.FindStringSubmatch(vb)
			es[entryPos].StartYear, _ = strconv.Atoi(x[1])
			es[entryPos].StartMonth, _ = strconv.Atoi(x[2])
			es[entryPos].StartDay, _ = strconv.Atoi(x[3])
			es[entryPos].StartHour, _ = strconv.Atoi(x[4])
			es[entryPos].StartMin, _ = strconv.Atoi(x[5])
			uploadEntry()
		case editing=="entries_end":
			r, _ := regexp.Compile("[ ]*([0-9]*)[ ]*/[ ]*([0-9]*)[ ]*/[ ]*([0-9]*)[ ]*([0-9]*)[ ]*:[ ]*([0-9]*)[ ]*")
			x := r.FindStringSubmatch(vb)
			es[entryPos].EndYear, _ = strconv.Atoi(x[1])
			es[entryPos].EndMonth, _ = strconv.Atoi(x[2])
			es[entryPos].EndDay, _ = strconv.Atoi(x[3])
			es[entryPos].EndHour, _ = strconv.Atoi(x[4])
			es[entryPos].EndMin, _ = strconv.Atoi(x[5])
			uploadEntry()
		case editing=="entries_category":
			es[entryPos].Category = vb
			uploadEntry()
		case editing=="entries_subcategory":
			es[entryPos].Subcategory = vb
			uploadEntry()
		case editing=="entries_info":
			es[entryPos].Info = vb
			uploadEntry()
	}

	if err = g.DeleteView("editing"); err != nil {
		return err
	}
	if v, err = g.SetCurrentView(editing); err != nil {
		return err
	}

	switch {
		case editing=="projects_name":
			refreshProjects(g,v)
		case editing=="projects_client":
			refreshProjects(g,v)
			g.SetCurrentView("client")
			_ = v.SetOrigin(0,0)
		default:
			refreshEntries(g,v)
		}

	return nil
}


func guiEsc(g *gocui.Gui, v *gocui.View) error {
	switch {
		case v.Name()=="projects_name":
		case v.Name()=="editing":
			if err := g.DeleteView("editing"); err != nil {
				return err
			}
			if _, err := g.SetCurrentView(editing); err != nil {
				return err
			}
		default:
			for _, i := range [...]string{"entries_rate", "entries_start", "entries_end", "entries_category", "entries_subcategory","entries_info","entries_status","entries_money"}   {
				if err := g.DeleteView(i); err != nil {
					return err
				}
			}
			if _, err := g.SetCurrentView("projects_name"); err != nil {
				return err
			}
	}
	return nil
}



func guiCreateEditProjectName(g *gocui.Gui, v *gocui.View) error {
	if(!verifyEditable(v)){
		return nil
	}

	maxX, maxY := g.Size()
	if v, err := g.SetView("editprojectname", maxX/2-30, maxY/2, maxX/2+30, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true

		if _, err := g.SetCurrentView("editprojectname"); err != nil {
			return err
		}
		fmt.Fprint(v,ps[projectPos].Name)
	}
	return nil
}

func guiDelEditProjectName(g *gocui.Gui, v *gocui.View) error {
	var err error
	_, cy := v.Cursor()
	vb, _ := v.Line(cy)
	ps[projectPos].Name = vb
	uploadProject()

	if err = g.DeleteView("editprojectname"); err != nil {
		return err
	}
	if v, err = g.SetCurrentView("projects_name"); err != nil {
		return err
	}
	printProjectsx(v)
	return nil
}

func guiCreateDeleteProject(g *gocui.Gui, v *gocui.View) error {
	if(!verifyEditable(v)){
		return nil
	}

	maxX, maxY := g.Size()
	if v, err := g.SetView("deleteproject", maxX/2-30, maxY/2, maxX/2+30, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true

		if _, err := g.SetCurrentView("deleteproject"); err != nil {
			return err
		}
		fmt.Fprint(v,"Confirm deletion")
	}
	return nil
}

func guiConfirmedDeleteProject(g *gocui.Gui, v *gocui.View) error {
	var err error

	deleteProject()

	if err = g.DeleteView("deleteproject"); err != nil {
		return err
	}
	if v, err = g.SetCurrentView("projects_name"); err != nil {
		return err
	}
	printProjectsx(v)
	return nil
}

func guiEscapeDeleteProject(g *gocui.Gui, v *gocui.View) error {

	if err := g.DeleteView("deleteproject"); err != nil {
		return err
	}
	if _, err := g.SetCurrentView("projects_name"); err != nil {
		return err
	}
	return nil
}


func guiDelete(g *gocui.Gui, v *gocui.View) error {
	if !verifyEditable(v) {
		return nil
	}

	editing = v.Name()
	if(editing=="clients"){
		return nil
	}

	maxX, maxY := g.Size()
	if v, err := g.SetView("delete", maxX/2-30, maxY/2, maxX/2+30, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true

		if _, err := g.SetCurrentView("delete"); err != nil {
			return err
		}
		fmt.Fprint(v,"Confirm deletion")
	}
	return nil
}

func guiDeleteConfirm(g *gocui.Gui, v *gocui.View) error {
	var err error

	if editing=="projects_name" {
		deleteProject()
	} else {
		deleteEntry()
	}

	if err = g.DeleteView("delete"); err != nil {
		return err
	}
	if v, err = g.SetCurrentView(editing); err != nil {
		return err
	}

	if editing=="projects_name" {
		printProjectsx(v)
	} else {
		refreshEntries(g,v)
	}
	return nil
}




func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyEsc, gocui.ModNone, guiEsc); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	if err := g.SetKeybinding("editing", gocui.KeyEnter, gocui.ModNone, guiEditConfirm); err != nil {
		return err
	}
	if err := g.SetKeybinding("delete", gocui.KeyEnter, gocui.ModNone, guiDeleteConfirm); err != nil {
		return err
	}

	if err := g.SetKeybinding("projects_name", gocui.KeyEnter, gocui.ModNone, selectProject); err != nil {
		return err
	}
	if err := g.SetKeybinding("projects_name", gocui.KeyCtrlN, gocui.ModNone, guiNewProject); err != nil {
		return err
	}

	for _, i := range [...]string{"entries_rate", "entries_start", "entries_end", "entries_category", "entries_subcategory","entries_info","entries_status"}   {
		if err := g.SetKeybinding(i, gocui.KeyCtrlN, gocui.ModNone, guiNewEntry); err != nil {
			return err
		}
	}

	for _, i := range [...]string{"clients","projects_name","projects_client","entries_rate", "entries_start", "entries_end", "entries_category", "entries_subcategory","entries_info","entries_status"}   {
		if err := g.SetKeybinding(i, gocui.KeyArrowLeft, gocui.ModNone, guiPreviousEntryView); err != nil {
			return err
		}
		if err := g.SetKeybinding(i, gocui.KeyArrowRight, gocui.ModNone, guiNextEntryView); err != nil {
			return err
		}
		if err := g.SetKeybinding(i, gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
			return err
		}
		if err := g.SetKeybinding(i, gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
			return err
		}
	}

	for _, i := range [...]string{"projects_name","projects_client","entries_rate", "entries_start", "entries_end", "entries_category", "entries_subcategory","entries_info","entries_status"}   {

		if err := g.SetKeybinding(i, gocui.KeySpace, gocui.ModNone, guiEdit); err != nil {
			return err
		}
		if err := g.SetKeybinding(i, gocui.KeyDelete, gocui.ModNone, guiDelete); err != nil {
			return err
		}
	}


	return nil
}

func saveMain(g *gocui.Gui, v *gocui.View) error {
	f, err := ioutil.TempFile("", "gocui_demo_")
	if err != nil {
		return err
	}
	defer f.Close()

	p := make([]byte, 5)
	v.Rewind()
	for {
		n, err := v.Read(p)
		if n > 0 {
			if _, err := f.Write(p[:n]); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func saveVisualMain(g *gocui.Gui, v *gocui.View) error {
	f, err := ioutil.TempFile("", "gocui_demo_")
	if err != nil {
		return err
	}
	defer f.Close()

	vb := v.ViewBuffer()
	if _, err := io.Copy(f, strings.NewReader(vb)); err != nil {
		return err
	}
	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("clients", -1, -1, 20, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Highlight = false
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		printClients(v)
	}
	if v, err := g.SetView("projects_name", 21, -1, maxX-20, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		printProjects_val(v,0)
	}
	if v, err := g.SetView("projects_client", maxX-20, -1, maxX, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		printProjects_val(v,1)
	}
	if v, err := g.SetView("status", -1, maxY-5, maxX, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		b, err := ioutil.ReadFile("Mark.Twain-Tom.Sawyer.txt")
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(v, "%s", b)
		v.Editable = true
		v.Wrap = true
		if _, err := g.SetCurrentView("projects_name"); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	loginInternal("r@rwhite.no", "hello")
	genPDF()

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Cursor = true
	g.InputEsc = true

	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
	fmt.Println(time.Now().Local())
}

func main1() {
	loginInternal("r@rwhite.no", "hello")
	genPDF()
	fmt.Println(time.Now().Local())
}
