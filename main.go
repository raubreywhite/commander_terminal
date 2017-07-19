package main

import (
	// Standard library packages
  "gopkg.in/mgo.v2/bson"
	"bytes"
	"encoding/json"
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sort"
	"time"
	"regexp"

	"github.com/jroimartin/gocui"
	"github.com/jung-kurt/gofpdf"

	"io"
	"io/ioutil"
	"log"
)

type (
	Bill struct {
		Year int
		Month int
		Billed string
		MoneyBilled int
		MoneyReceived int
	}

	// User represents the structure of our resource
	User struct {
		Id     bson.ObjectId `json:"id" bson:"_id"`
		Session     string `json:"session" bson:"session"`
		Email  string        `json:"email" bson:"email"`
		Name   string        `json:"name" bson:"name"`
		Password    string           `json:"password" bson:"password"`
		Projects    []Project           `json:"projects" bson:"projects"`
		Success bool `json:"success" bson:"success"`
		LoggedIn bool `json:"loggedin" bson:"loggedin"`
	}

	UserSession struct {
		Id     bson.ObjectId `json:"id" bson:"_id"`
		Session     string `json:"session" bson:"session"`
		Success bool `json:"success"`
	}

	Project struct {
		Name   string        `json:"name" bson:"name"`
		Client   string        `json:"client" bson:"client"`
		Entries    []Entry           `json:"entries" bson:"entries"`
		Bills []Bill `json:"bills" bson:"bills"`
	}

	Entry struct {
		Session     string `json:"session" bson:"session"`
		Status   string        `json:"status" bson:"status"`
		Rate   int        `json:"rate" bson:"rate"`
		StartYear   int        `json:"startYear" bson:"startYear"`
		StartMonth   int        `json:"startMonth" bson:"startMonth"`
		StartDay   int        `json:"startDay" bson:"startDay"`
		StartHour   int        `json:"startHour" bson:"startHour"`
		StartMin   int        `json:"startMin" bson:"startMin"`
		EndYear   int        `json:"endYear" bson:"endYear"`
		EndMonth   int        `json:"endMonth" bson:"endMonth"`
		EndDay   int        `json:"endDay" bson:"endDay"`
		EndHour   int        `json:"endHour" bson:"endHour"`
		EndMin   int        `json:"endMin" bson:"endMin"`
		Category   string        `json:"category" bson:"category"`
		Subcategory   string        `json:"Subcategory" bson:"Subcategory"`
		Info   string        `json:"name" bson:"name"`
	}
)


var email = "r@rwhite.no"
var password = "hello"

var u = User{LoggedIn:false}
var ps = []Project{}
var es = []Entry{}
var level = "main"
var editing = "nothing"
var client = "Unknown"
var cs = []string{}
var psGivenClient = []int{}
var bsGivenProject = []int{}


var clientPos = 0
var projectPos = 0
var entryPos = 0
var billPos = 0



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

// WORKING WITH DATABASE

func writeUser(){

	outputjson,_:=json.Marshal(u)

	// delete file
	_ = os.Remove("output.json")
	
  //open the file
  myfile,errorprocess := os.OpenFile("output.json",os.O_WRONLY|os.O_CREATE,0666)
  defer myfile.Close()
  //check for errors
  if errorprocess!=nil{
    fmt.Println("Oops, there is an error while opening the file")
  }
 
  //define the 'string writer'
  filewriter:=bufio.NewWriter(myfile)
 
  //write the JSON string. First we need to convert the outputjson to string, and then write it
  filewriter.WriteString(string(outputjson))
 
  //you know what to do
  filewriter.Flush()
}

func readUser() bool{
  //open the file
  myfile,err := os.OpenFile("output.json",os.O_RDONLY,0666)
  defer myfile.Close()
  //check for errors
  if err!=nil{
    fmt.Println("Oops, there is an error while opening the file")
		return false
  }

	jsonParser := json.NewDecoder(myfile)
	jsonParser.Decode(&u)
	fmt.Println(u)
	return true
}

func createUser() {
	u.Email = email
	u.Password = password

	jsonStr, err := json.Marshal(u)
	req, err := http.NewRequest("POST", "http://10.0.1.10:8888/create/users", bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&u)

	if u.Success {
		downloadUser()
		writeUser()
	}
}

func uploadUser() {
	ProcessProjectsBills()

	jsonStr, err := json.Marshal(u)
	req, err := http.NewRequest("POST", "http://10.0.1.10:8888/edit/users", bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&u)

	establishProjectsGivenClient()

	writeUser()
}

func downloadUser(){
	jsonStr, err := json.Marshal(u)
	req, err := http.NewRequest("POST", "http://10.0.1.10:8888/login", bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&u)

	if u.LoggedIn {
		establishProjectsGivenClient()
		ProcessProjectsBills()
		establishBillsGivenProject()
	}
}

func createProject() {
	bills := []Bill{}
	currentYear, _, _, _, _ := GetIntsNow()

	for y := currentYear-1; y<=currentYear+5; y++ {
		for m := 1; m<=12; m++ {
			b:=Bill{y, m, "", 0, 0}
			bills = append(bills, b)
		}
	}
	p := Project{Name: "A new project", Client: "Unknown", Bills: bills}
	u.Projects = append(u.Projects, p)
	uploadUser()
}

func createEntry(arg string) {
	currentYear, currentMonth, currentDay, currentHour, currentMin := GetIntsNow()
	endYear, endMonth, endDay, endHour, endMin := IncreaseInts(currentYear, currentMonth, currentDay, currentHour, currentMin, 60)

	e := Entry{Session: u.Session,
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

	u.Projects[projectPos].Entries = append(u.Projects[projectPos].Entries, e)

	uploadUser()
}

func deleteEntry(){
	u.Projects[projectPos].Entries = append(u.Projects[projectPos].Entries[:entryPos], u.Projects[projectPos].Entries[entryPos+1:]...)

	uploadUser()
}

func deleteProject(){
	u.Projects = append(u.Projects[:projectPos],u.Projects[projectPos+1:]...)

	uploadUser()
}



// CONVERT DATA TO USEFUL THINGS TO DISPLAY

func EntryToRowInternal(e Entry) []string {
	rate := strconv.Itoa(e.Rate)
	if rate=="" {
		rate= " "
	}

	startTime := ConvertIntsToString(e.StartYear, e.StartMonth, e.StartDay, e.StartHour, e.StartMin)

	endTime := ConvertIntsToString(e.EndYear, e.EndMonth, e.EndDay, e.EndHour, e.EndMin)

	category := e.Category

	subcategory := e.Subcategory

	info := e.Info

	status := e.Status
	if status=="" {
		status = " "
	}

	startDate := ConvertIntsToStringDate(e.StartYear, e.StartMonth, e.StartDay)

	duration := ConvertIntsToTime(e.EndYear, e.EndMonth, e.EndDay, e.EndHour, e.EndMin).Sub(ConvertIntsToTime(e.StartYear, e.StartMonth, e.StartDay, e.StartHour, e.StartMin)).Hours()

	money := strconv.FormatFloat(float64(e.Rate)*duration,'f', 0, 64)

	return []string{
		rate,
		startTime,
		endTime,
		category,
		subcategory,
		info,
		money,
		startDate,
		strconv.FormatFloat(duration,'f', 2, 64)}
}


func EntryToRow() []string {
	e := u.Projects[projectPos].Entries[entryPos]

	return EntryToRowInternal(e)
}

func BillToRow() []string {

	b := u.Projects[projectPos].Bills[billPos]

	return []string{
		b.Billed,
		strconv.Itoa(b.Year)+"-"+strconv.Itoa(b.Month),
		strconv.Itoa(b.MoneyBilled),
		strconv.Itoa(b.MoneyReceived)}
}

func ProjectToRow() []string {

	client := u.Projects[projectPos].Client
	name := u.Projects[projectPos].Name

	return []string{
		name,
		client}
}

// GRAPHICS

func printClients(v *gocui.View) error {
	v.Clear()
	if len(cs) == 0 {

	} else {
		for index, c := range cs {
			star := ""
			if index == clientPos {
				star = "** "
			}
			fmt.Fprintf(v,"%v%v\n",star,c)
		}
	}
	return nil
}

func printProjects_val(v *gocui.View, pos int) error {
	origProjectPos := projectPos
	v.Clear()
	if len(u.Projects) == 0 {

	} else {
		for _, index := range psGivenClient {
			projectPos=index
			x := ProjectToRow()

			star := ""
			if projectPos == origProjectPos {
				star = "** "
			}
			fmt.Fprintf(v,"%v%v\n",star,x[pos])
			}
	}
	projectPos = origProjectPos
	return nil
}

func printEntries_val(v *gocui.View, pos int) error {
	v.Clear()
	if len(u.Projects[projectPos].Entries) == 0 {

	} else {
		for index, _ := range u.Projects[projectPos].Entries {
			entryPos=index
			x := EntryToRow()
			fmt.Fprintf(v,"%v\n",x[pos])
			}
	}
	return nil
}

func printBills_val(v *gocui.View, pos int) error {

	v.Clear()
	if len(bsGivenProject) == 0 {

	} else {
		for _, index := range bsGivenProject {
			billPos = index
			x := BillToRow()

			fmt.Fprintf(v,"%v\n",x[pos])
		}

	}
	return nil
}


func ProcessProjectBill(pos int){
	for i, _ := range u.Projects[pos].Bills {
		u.Projects[pos].Bills[i].MoneyBilled = 0
		money := 0.0
		for _ , e := range u.Projects[pos].Entries {
			if u.Projects[pos].Bills[i].Year == e.StartYear && u.Projects[pos].Bills[i].Month == e.StartMonth {
				temp := EntryToRowInternal(e)
				tempMoney, _ := strconv.ParseFloat(temp[6],64)
				money += tempMoney
			}
			u.Projects[pos].Bills[i].MoneyBilled = int(money)
		}
	}
}

func ProcessProjectsBills(){
	for i, _ := range u.Projects {
		ProcessProjectBill(i)
	}
}

func establishBillsGivenProject(){

	bsGivenProject = []int{}

	// create client list
	for i, b := range u.Projects[projectPos].Bills {
		if b.MoneyBilled > 0.0 {
			bsGivenProject = append(bsGivenProject, i)
		}
	}
}

func establishProjectsGivenClient(){
	if len(u.Projects)==0 {
		clientPos = 0
		createProject()
	}
	//c = "Unknown"
	cs = []string{}
	psGivenClient = []int{}

	// create client list
	for _, p := range u.Projects {
		unique := true
		for _, x := range cs {
			if x == p.Client {
				unique = false
			}
		}
		if unique {
			cs = append(cs, p.Client)
		}
	}
	sort.Strings(cs)
	for clientPos >= len(cs) {
		clientPos--
	}
	client = cs[clientPos]
	// create project list
	for i, p := range u.Projects {
		if client == p.Client {
			psGivenClient = append(psGivenClient, i)
		}
	}

}





func refreshProjects(g *gocui.Gui, v *gocui.View) error {
	var err error
	original := v.Name()
	establishProjectsGivenClient()
	establishBillsGivenProject()

	v, err = g.SetCurrentView("clients")
	printClients(v)

	for i, n := range [...]string{"projects_name", "projects_client"}   {
		v, err = g.SetCurrentView(n)
		printProjects_val(v,i)
	}

	for i, n := range [...]string{"projects_billed" ,"projects_month","projects_moneybilled","projects_moneyreceived"} {
		v, err = g.SetCurrentView(n)
		printBills_val(v,i)
	}
	v, err = g.SetCurrentView(original)
	return err
}

func refreshEntries(g *gocui.Gui, v *gocui.View) error {
	var err error
	original := v.Name()
	for i, n := range [...]string{"entries_rate", "entries_start", "entries_end", "entries_category", "entries_subcategory","entries_info","entries_money"}   {
		v, err = g.SetCurrentView(n)
		printEntries_val(v,i)
	}
	v, err = g.SetCurrentView(original)
	return err
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
	for pIndex, p := range u.Projects {
		projectPos = pIndex

		for _, e := range u.Projects[projectPos].Entries {
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
  fmt.Println("GENPDF")
  fmt.Println(yearMonths)
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
      fmt.Println(ymProjectPos)

			projectPos = ymProjectPos.ProjectPos

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
			for index, e := range u.Projects[projectPos].Entries {
				entryPos=index
				if ymProjectPos.Year != e.StartYear {
					continue
				}
				if ymProjectPos.Month != e.StartMonth {
					continue
				}
				x := EntryToRow()
				fmt.Println(x)
				temp, _ := strconv.Atoi(x[6])
				billed = billed + temp
				temp2, _ := strconv.ParseFloat(x[8], 64)
				hours = hours + temp2

				pdf.CellFormat(w[0], 6, x[7], "L", 0, "C", false, 0, "")
				pdf.CellFormat(w[1], 6, x[8], "", 0, "C", false, 0, "")
				pdf.CellFormat(w[2], 6, x[0], "", 0, "C", false, 0, "")
				pdf.CellFormat(w[3], 6, x[6], "", 0, "C", false, 0, "")
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


func loginInternal(email string, password string) {
	u = User{Email: email, Password: password}
	downloadUser()
}

func selectProject(g *gocui.Gui, v *gocui.View) error {
	_, cy := v.Cursor()
	if(cy<len(u.Projects)){
		projectPos=psGivenClient[cy]
		return displayEntries(g,v)
	}
	return nil
}

func guiNewProject(g *gocui.Gui, v *gocui.View) error {
	createProject()
	refreshProjects(g,v)
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
		cx, cy := v.Cursor()
		for cy >= len(cs){
			v.SetCursor(cx, cy-1)
			cy = cy - 1
		}
		clientPos = cy
		refreshProjects(g,v)
	}
	if v.Name()=="projects_name" || v.Name()=="projects_client" {
		cx, cy := v.Cursor()
		for cy >= len(psGivenClient){
			v.SetCursor(cx, cy-1)
			cy = cy - 1
		}
		projectPos = psGivenClient[cy]
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
		cx, cy := v.Cursor()
		for cy >= len(cs){
			v.SetCursor(cx, cy-1)
			cy = cy - 1
		}
		clientPos = cy
		refreshProjects(g,v)
	}
	if v.Name()=="projects_name" || v.Name()=="projects_client" {
		cx, cy := v.Cursor()
		for cy >= len(psGivenClient){
			v.SetCursor(cx, cy-1)
			cy = cy - 1
		}
		projectPos = psGivenClient[cy]
		refreshProjects(g,v)
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
			v, err = g.SetCurrentView("entries_rate")
		case v.Name()=="clients":
			v, err = g.SetCurrentView("projects_name")
		case v.Name()=="projects_name":
			v, err = g.SetCurrentView("projects_client")
		case v.Name()=="projects_client":
			v, err = g.SetCurrentView("projects_billed")
		case v.Name()=="projects_billed":
			v, err = g.SetCurrentView("projects_month")
		case v.Name()=="projects_month":
			v, err = g.SetCurrentView("projects_moneybilled")
		case v.Name()=="projects_moneybilled":
			v, err = g.SetCurrentView("projects_moneyreceived")
		case v.Name()=="projects_moneyreceived":
			v, err = g.SetCurrentView("clients")
	}
	v.Highlight = true
	v.SetCursor(cx, cy)

	if v.Name()=="clients"{
		cx, cy := v.Cursor()
		for cy >= len(cs){
			v.SetCursor(cx, cy-1)
			cy = cy - 1
		}
		clientPos = cy
		refreshProjects(g,v)
	}
	if v.Name()=="projects_name" || v.Name()=="projects_client" {
		cx, cy := v.Cursor()
		for cy >= len(psGivenClient){
			v.SetCursor(cx, cy-1)
			cy = cy - 1
		}
		projectPos = psGivenClient[cy]
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
			v, err = g.SetCurrentView("entries_info")
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
		case v.Name()=="clients":
			v, err = g.SetCurrentView("projects_moneyreceived")
		case v.Name()=="projects_name":
			v, err = g.SetCurrentView("clients")
		case v.Name()=="projects_client":
			v, err = g.SetCurrentView("projects_name")
		case v.Name()=="projects_billed":
			v, err = g.SetCurrentView("projects_client")
		case v.Name()=="projects_month":
			v, err = g.SetCurrentView("projects_billed")
		case v.Name()=="projects_moneybilled":
			v, err = g.SetCurrentView("projects_month")
		case v.Name()=="projects_moneyreceived":
			v, err = g.SetCurrentView("projects_moneybilled")
	}
	v.Highlight = true
	v.SetCursor(cx, cy)

	if v.Name()=="clients"{
		cx, cy := v.Cursor()
		for cy >= len(cs){
			v.SetCursor(cx, cy-1)
			cy = cy - 1
		}
		clientPos = cy
		refreshProjects(g,v)
	}
	if v.Name()=="projects_name" || v.Name()=="projects_client" {
		cx, cy := v.Cursor()
		for cy >= len(psGivenClient){
			v.SetCursor(cx, cy-1)
			cy = cy - 1
		}
		projectPos = psGivenClient[cy]
		refreshProjects(g,v)
	}
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

	if v, err = g.SetView("entries_info", 71, -1, maxX-8, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		printEntries_val(v,5)
		//v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
	}


	if v, err = g.SetView("entries_money", maxX-8, -1, maxX, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		printEntries_val(v,6)
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
		case v.Name()=="projects_name" || v.Name()=="projects_client":
			if(cy<len(psGivenClient)){
				projectPos=psGivenClient[cy]
				return true
			} else {
				return false
			}
		default:
			if(cy<len(u.Projects[projectPos].Entries)){
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
		case u.Projects[projectPos].Entries[entryPos].Status=="":
			u.Projects[projectPos].Entries[entryPos].Status="B"
		case u.Projects[projectPos].Entries[entryPos].Status=="B":
			u.Projects[projectPos].Entries[entryPos].Status="+"
		case u.Projects[projectPos].Entries[entryPos].Status=="+":
			u.Projects[projectPos].Entries[entryPos].Status=""
		}
		//es[entryPos].Status=""
		uploadUser()
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
			fmt.Fprint(v,u.Projects[projectPos].Name)
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
	vb = strings.Replace(vb, "\u0000", "", -1)

	switch {
		case editing=="projects_name":
			u.Projects[projectPos].Name = vb
		case editing=="projects_client":
			u.Projects[projectPos].Client = vb
		case editing=="entries_rate":
			u.Projects[projectPos].Entries[entryPos].Rate, _ = strconv.Atoi(vb)
		case editing=="entries_start":
			r, _ := regexp.Compile("[ ]*([0-9]*)[ ]*/[ ]*([0-9]*)[ ]*/[ ]*([0-9]*)[ ]*([0-9]*)[ ]*:[ ]*([0-9]*)[ ]*")
			x := r.FindStringSubmatch(vb)
			u.Projects[projectPos].Entries[entryPos].StartYear, _ = strconv.Atoi(x[1])
			u.Projects[projectPos].Entries[entryPos].StartMonth, _ = strconv.Atoi(x[2])
			u.Projects[projectPos].Entries[entryPos].StartDay, _ = strconv.Atoi(x[3])
			u.Projects[projectPos].Entries[entryPos].StartHour, _ = strconv.Atoi(x[4])
			u.Projects[projectPos].Entries[entryPos].StartMin, _ = strconv.Atoi(x[5])
		case editing=="entries_end":
			r, _ := regexp.Compile("[ ]*([0-9]*)[ ]*/[ ]*([0-9]*)[ ]*/[ ]*([0-9]*)[ ]*([0-9]*)[ ]*:[ ]*([0-9]*)[ ]*")
			x := r.FindStringSubmatch(vb)
			u.Projects[projectPos].Entries[entryPos].EndYear, _ = strconv.Atoi(x[1])
			u.Projects[projectPos].Entries[entryPos].EndMonth, _ = strconv.Atoi(x[2])
			u.Projects[projectPos].Entries[entryPos].EndDay, _ = strconv.Atoi(x[3])
			u.Projects[projectPos].Entries[entryPos].EndHour, _ = strconv.Atoi(x[4])
			u.Projects[projectPos].Entries[entryPos].EndMin, _ = strconv.Atoi(x[5])
		case editing=="entries_category":
			u.Projects[projectPos].Entries[entryPos].Category = vb
		case editing=="entries_subcategory":
			u.Projects[projectPos].Entries[entryPos].Subcategory = vb
		case editing=="entries_info":
			u.Projects[projectPos].Entries[entryPos].Info = vb
	}
	uploadUser()

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
			for _, i := range [...]string{"entries_rate", "entries_start", "entries_end", "entries_category", "entries_subcategory","entries_info","entries_money"}   {
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

	if editing=="projects_name" || editing=="projects_client" {
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

	if editing=="projects_name" || editing=="projects_client" {
		refreshProjects(g,v)
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

	for _, i := range [...]string{"clients","projects_name","projects_client","projects_billed","projects_month","projects_moneybilled","projects_moneyreceived","entries_rate", "entries_start", "entries_end", "entries_category", "entries_subcategory","entries_info","entries_status"}   {
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

	}
	if v, err := g.SetView("projects_name", 21, -1, maxX-52, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Highlight = false
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

	}
	if v, err := g.SetView("projects_client", maxX-52, -1, maxX-32, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Highlight = false
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

	}
	if v, err := g.SetView("projects_billed", maxX-32, -1, maxX-30, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Highlight = false
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

	}
	if v, err := g.SetView("projects_month", maxX-30, -1, maxX-20, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Highlight = false
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

	}
	if v, err := g.SetView("projects_moneybilled", maxX-20, -1, maxX-10, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Highlight = false
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

	}
	if v, err := g.SetView("projects_moneyreceived", maxX-10, -1, maxX, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Highlight = false
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

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

		refreshProjects(g,v)
		if _, err := g.SetCurrentView("projects_name"); err != nil {
			return err
		}


	}



	return nil
}

func main() {
	if readUser() {
		loginInternal(u.Email, u.Password)
		if u.LoggedIn == false {
			readUser()
			createUser()
		}
	} else {
		createUser()
	}

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

func mainx() {
	if readUser() {
		loginInternal(u.Email, u.Password)
		if u.LoggedIn == false {
			readUser()
			createUser()
		}
	} else {
		createUser()
	}

	genPDF()
}

