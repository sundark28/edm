package main

import (
	"database/sql"
	"edm/internal/mail"
	"edm/pkg/accs"
	"edm/pkg/datetime"
	"edm/pkg/memdb"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alecxcode/sqla"
)

// Task describes a task to perform
type Task struct {
	//sql generate
	ID           int
	Created      datetime.DateTime `sql-gen:"bigint,IDX"`
	PlanStart    datetime.DateTime `sql-gen:"bigint,IDX"`
	PlanDue      datetime.DateTime `sql-gen:"bigint,IDX"`
	StatusSet    datetime.DateTime `sql-gen:"bigint,IDX"`
	Creator      *Profile          `sql-gen:"FK_NULL,FK_NOACTION"`
	Assignee     *Profile          `sql-gen:"FK_NULL,FK_NOACTION"`
	Participants []int             `sql-gen:"varchar(4000)"`
	Topic        string            `sql-gen:"varchar(255)"`
	Content      string            `sql-gen:"varchar(max)"`
	TaskStatus   int
	Project      int      //this is for future use
	FileList     []string `sql-gen:"varchar(max)"`
}

func (t Task) print() {
	fmt.Printf("%#v\n", t)
}

// GiveStatus executes in a template to deliver the status of a task
func (t Task) GiveStatus(stslice []string, unknown string) string {
	if t.TaskStatus < len(stslice) && t.TaskStatus >= Undefined {
		return stslice[t.TaskStatus]
	} else {
		return unknown
	}
}

// GiveCreatorID executes in a template to deliver the creator ID of this object
func (t Task) GiveCreatorID() int {
	if t.Creator == nil {
		return 0
	} else {
		return t.Creator.ID
	}
}

// GiveAssigneeID executes in a template to deliver the assignee ID of this object
func (t Task) GiveAssigneeID() int {
	if t.Assignee == nil {
		return 0
	} else {
		return t.Assignee.ID
	}
}

// GiveDateTime executes in a template to deliver the queried date and time of a task
func (t Task) GiveDateTime(dateWhat string, dateFmt string, timeFmt string, sep string) string {

	var dt datetime.DateTime
	var rt string

	switch dateWhat {
	case "Created":
		dt = t.Created
	case "PlanStart":
		dt = t.PlanStart
	case "PlanDue":
		dt = t.PlanDue
	case "StatusSet":
		dt = t.StatusSet
	default:
		return "wrong arg"
	}

	if dt.Day == 0 {
		return ""
	}

	if timeFmt == "12h am/pm" {
		rt = datetime.TimeToString12(dt.Hour, dt.Minute)
	} else if timeFmt == "24h" {
		rt = datetime.TimeToString24(dt.Hour, dt.Minute)
	} else {
		rt = datetime.TimeToString24(dt.Hour, dt.Minute)
	}

	return datetime.DateToString(datetime.Date{Year: dt.Year, Month: dt.Month, Day: dt.Day}, dateFmt) + sep + rt
}

// GiveShortFileName executes in a template to deliver the shortened file name
func (t Task) GiveShortFileName(index int) string {
	if index >= len(t.FileList) {
		return ""
	} else {
		return string([]rune(t.FileList[index])[0]) + ".." + filepath.Ext(t.FileList[index])
	}
}

func (t *Task) create(db *sql.DB, DBType byte) (lastid int, rowsaff int) {
	var args sqla.AnyTslice
	if t.Created.Day != 0 {
		args = args.AppendInt64("Created", datetime.DateTimeToInt64(t.Created))
	}
	if t.PlanStart.Day != 0 {
		args = args.AppendInt64("PlanStart", datetime.DateTimeToInt64(t.PlanStart))
	}
	if t.PlanDue.Day != 0 {
		args = args.AppendInt64("PlanDue", datetime.DateTimeToInt64(t.PlanDue))
	}
	if t.StatusSet.Day != 0 {
		args = args.AppendInt64("StatusSet", datetime.DateTimeToInt64(t.StatusSet))
	}
	if t.Creator != nil {
		args = args.AppendInt("Creator", t.Creator.ID)
	}
	if t.Assignee != nil {
		args = args.AppendInt("Assignee", t.Assignee.ID)
	}
	args = args.AppendNonEmptyString("Topic", t.Topic)
	args = args.AppendNonEmptyString("Content", t.Content)
	args = args.AppendInt("TaskStatus", t.TaskStatus)
	args = args.AppendJSONList("FileList", t.FileList)
	lastid, rowsaff = sqla.InsertObject(db, DBType, "tasks", args)
	return lastid, rowsaff
}

func (t *Task) update(db *sql.DB, DBType byte) (rowsaff int) {
	var args sqla.AnyTslice
	if t.PlanStart.Day != 0 {
		args = args.AppendInt64("PlanStart", datetime.DateTimeToInt64(t.PlanStart))
	} else {
		args = args.AppendNil("PlanStart")
	}
	if t.PlanDue.Day != 0 {
		args = args.AppendInt64("PlanDue", datetime.DateTimeToInt64(t.PlanDue))
	} else {
		args = args.AppendNil("PlanDue")
	}
	if t.StatusSet.Day != 0 {
		args = args.AppendInt64("StatusSet", datetime.DateTimeToInt64(t.StatusSet))
	}
	if t.Assignee != nil {
		args = args.AppendInt("Assignee", t.Assignee.ID)
	} else {
		args = args.AppendNil("Assignee")
	}
	args = args.AppendStringOrNil("Topic", t.Topic)
	args = args.AppendStringOrNil("Content", t.Content)
	args = args.AppendInt("TaskStatus", t.TaskStatus)
	args = args.AppendJSONList("FileList", t.FileList)
	rowsaff = sqla.UpdateObject(db, DBType, "tasks", args, t.ID)
	return rowsaff
}

func (t *Task) updateStatus(db *sql.DB, DBType byte) (rowsaff int) {
	var args sqla.AnyTslice
	args = args.AppendInt64("StatusSet", datetime.DateTimeToInt64(t.StatusSet))
	args = args.AppendInt("TaskStatus", t.TaskStatus)
	rowsaff = sqla.UpdateObject(db, DBType, "tasks", args, t.ID)
	return rowsaff
}

func (t *Task) load(db *sql.DB, DBType byte) error {

	row := db.QueryRow(`SELECT
tasks.ID, Created, PlanStart, PlanDue, StatusSet, Creator, Assignee, Participants, Topic, Content, TaskStatus, FileList,
creator.ID, creator.FirstName, creator.Surname, creator.JobTitle, creator.Contacts, creator.UserLock,
assignee.ID, assignee.FirstName, assignee.Surname, assignee.JobTitle, assignee.Contacts, assignee.UserLock
FROM tasks
LEFT JOIN profiles creator ON creator.ID = Creator
LEFT JOIN profiles assignee ON assignee.ID = Assignee
WHERE tasks.ID = `+sqla.MakeParam(DBType, 1), t.ID)

	var Created sql.NullInt64
	var PlanStart sql.NullInt64
	var PlanDue sql.NullInt64
	var StatusSet sql.NullInt64
	var Creator sql.NullInt64
	var Assignee sql.NullInt64
	var Participants sql.NullString
	var Topic sql.NullString
	var Content sql.NullString
	var TaskStatus sql.NullInt64
	var FileList sql.NullString

	var CreatorID sql.NullInt64
	var CreatorFirstName sql.NullString
	var CreatorSurname sql.NullString
	var CreatorJobTitle sql.NullString
	var CreatorContacts sql.NullString
	var CreatorUserLock sql.NullInt64

	var AssigneeID sql.NullInt64
	var AssigneeFirstName sql.NullString
	var AssigneeSurname sql.NullString
	var AssigneeJobTitle sql.NullString
	var AssigneeContacts sql.NullString
	var AssigneeUserLock sql.NullInt64

	err := row.Scan(&t.ID, &Created, &PlanStart, &PlanDue, &StatusSet, &Creator, &Assignee, &Participants, &Topic, &Content, &TaskStatus, &FileList,
		&CreatorID, &CreatorFirstName, &CreatorSurname, &CreatorJobTitle, &CreatorContacts, &CreatorUserLock,
		&AssigneeID, &AssigneeFirstName, &AssigneeSurname, &AssigneeJobTitle, &AssigneeContacts, &AssigneeUserLock)
	if err != nil {
		return err
	}

	t.Created = datetime.Int64ToDateTime(Created.Int64)
	t.PlanStart = datetime.Int64ToDateTime(PlanStart.Int64)
	t.PlanDue = datetime.Int64ToDateTime(PlanDue.Int64)
	t.StatusSet = datetime.Int64ToDateTime(StatusSet.Int64)
	if CreatorID.Valid == true {
		t.Creator = &Profile{
			ID:        int(CreatorID.Int64),
			FirstName: CreatorFirstName.String,
			Surname:   CreatorSurname.String,
			JobTitle:  CreatorJobTitle.String,
			Contacts:  unmarshalNonEmptyProfileContacts(CreatorContacts.String),
			UserLock:  int(CreatorUserLock.Int64),
		}
	} else {
		t.Creator = nil
	}
	if AssigneeID.Valid == true {
		t.Assignee = &Profile{
			ID:        int(AssigneeID.Int64),
			FirstName: AssigneeFirstName.String,
			Surname:   AssigneeSurname.String,
			JobTitle:  AssigneeJobTitle.String,
			Contacts:  unmarshalNonEmptyProfileContacts(AssigneeContacts.String),
			UserLock:  int(AssigneeUserLock.Int64),
		}
	} else {
		t.Assignee = nil
	}

	t.Participants = sqla.UnmarshalNonEmptyJSONListInt(Participants.String)
	t.Topic = Topic.String
	t.Content = Content.String
	t.TaskStatus = int(TaskStatus.Int64)
	t.FileList = sqla.UnmarshalNonEmptyJSONList(FileList.String)

	return nil
}

func (t *Task) loadParticipants(db *sql.DB, DBType byte) (ProfList []Profile, err error) {

	if len(t.Participants) > 0 {
		var sq = "SELECT ID, FirstName, Surname, JobTitle, Contacts, UserLock FROM profiles "
		var args, argstoAppend []interface{}

		_, sq, argstoAppend = sqla.BuildSQLIN(DBType, sq, 0, "ID", t.Participants)
		args = append(args, argstoAppend...)

		if DEBUG {
			log.Println(sq, t.Participants)
		}
		rows, err := db.Query(sq, args...)
		if err != nil {
			return ProfList, err
		}
		defer rows.Close()

		var FirstName sql.NullString
		var Surname sql.NullString
		var JobTitle sql.NullString
		var Contacts sql.NullString
		var UserLock sql.NullInt64

		for rows.Next() {
			var p Profile
			err = rows.Scan(&p.ID, &FirstName, &Surname, &JobTitle, &Contacts, &UserLock)
			if err != nil {
				return ProfList, err
			}
			p.FirstName = FirstName.String
			p.Surname = Surname.String
			p.JobTitle = JobTitle.String
			p.Contacts = unmarshalNonEmptyProfileContacts(Contacts.String)
			p.UserLock = int(UserLock.Int64)
			ProfList = append(ProfList, p)
		}

	}
	return ProfList, nil
}

// Comment is a discussion item to a task
type Comment struct {
	//sql generate
	ID       int
	Created  datetime.DateTime `sql-gen:"bigint"`
	Creator  *Profile          `sql-gen:"FK_NULL"`
	Task     *Task             `sql-gen:"IDX,FK_CASCADE"`
	Content  string            `sql-gen:"varchar(max)"`
	FileList []string          `sql-gen:"varchar(max)"`
}

func (c *Comment) create(db *sql.DB, DBType byte) (lastid int, rowsaff int) {
	var args sqla.AnyTslice
	if c.Created.Day != 0 {
		args = args.AppendInt64("Created", datetime.DateTimeToInt64(c.Created))
	}
	args = args.AppendInt("Creator", c.Creator.ID)
	args = args.AppendInt("Task", c.Task.ID)
	args = args.AppendNonEmptyString("Content", c.Content)
	args = args.AppendJSONList("FileList", c.FileList)
	lastid, rowsaff = sqla.InsertObject(db, DBType, "comments", args)
	return lastid, rowsaff
}

// GiveCreatorID executes in a template to deliver the creator ID of this object
func (c Comment) GiveCreatorID() int {
	if c.Creator == nil {
		return 0
	} else {
		return c.Creator.ID
	}
}

// Inci executes in a template to deliver value increment of +1
func (c Comment) Inci(n int) int {
	return n + 1
}

// GiveDateTime executes in a template to deliver the queried date and time of a comment
func (c Comment) GiveDateTime(dateFmt string, timeFmt string, sep string) string {
	var dt = c.Created
	var rt string
	if timeFmt == "12h am/pm" {
		rt = datetime.TimeToString12(dt.Hour, dt.Minute)
	} else if timeFmt == "24h" {
		rt = datetime.TimeToString24(dt.Hour, dt.Minute)
	} else {
		rt = datetime.TimeToString24(dt.Hour, dt.Minute)
	}
	if dt.Day == 0 {
		return ""
	}
	return datetime.DateToString(datetime.Date{Year: dt.Year, Month: dt.Month, Day: dt.Day}, dateFmt) + sep + rt
}

func (t *Task) loadComments(db *sql.DB, DBType byte) (CommList []Comment, err error) {

	rows, err := db.Query(`SELECT c.ID, c.Created, c.Creator, c.Content, c.FileList,
p.ID, p.FirstName, p.Surname,  p.JobTitle
FROM comments c
LEFT JOIN profiles p ON p.ID = c.Creator
WHERE Task = `+sqla.MakeParam(DBType, 1)+` ORDER BY c.Created ASC, c.ID ASC`, t.ID)
	if err != nil {
		return CommList, err
	}
	defer rows.Close()

	var Created sql.NullInt64
	var Creator sql.NullInt64
	var Content sql.NullString
	var FileList sql.NullString

	var CreatorID sql.NullInt64
	var CreatorFirstName sql.NullString
	var CreatorSurname sql.NullString
	var CreatorJobTitle sql.NullString

	for rows.Next() {
		var c Comment
		err = rows.Scan(&c.ID, &Created, &Creator, &Content, &FileList,
			&CreatorID, &CreatorFirstName, &CreatorSurname, &CreatorJobTitle)
		if err != nil {
			return CommList, err
		}
		c.Created = datetime.Int64ToDateTime(Created.Int64)
		if Creator.Valid {
			c.Creator = &Profile{
				ID:        int(CreatorID.Int64),
				FirstName: CreatorFirstName.String,
				Surname:   CreatorSurname.String,
				JobTitle:  CreatorJobTitle.String,
			}
		}
		c.Content = Content.String
		c.FileList = sqla.UnmarshalNonEmptyJSONList(FileList.String)
		CommList = append(CommList, c)
	}

	return CommList, nil
}

// TaskPage is passed into template
type TaskPage struct {
	AppTitle       string
	AppVersion     string
	PageTitle      string
	LoggedinID     int
	UserConfig     UserConfig
	Task           Task      //payload
	Comments       []Comment //payload
	Participants   []Profile
	Message        string
	RemoveAllowed  bool
	Editable       bool
	IamAssignee    bool
	IamParticipant bool
	New            bool
	TaskStatuses   []string
	UserList       []memdb.ObjHasID
}

func (bs *BaseStruct) taskHandler(w http.ResponseWriter, r *http.Request) {

	const (
		CREATED = iota
		ASSIGNED
		INPROGRESS
		STUCK
		DONE
		CANCELLED
	)
	const ADMIN = 1

	allow, id := bs.authVerify(w, r)
	if !allow {
		return
	}

	if bs.validURLs.Task.FindStringSubmatch(r.URL.Path) == nil {
		http.NotFound(w, r)
		return
	}

	var err error

	var Page = TaskPage{
		AppTitle:       bs.text.AppTitle,
		AppVersion:     AppVersion,
		LoggedinID:     id,
		Editable:       false,
		IamAssignee:    false,
		IamParticipant: false,
		New:            false,
		TaskStatuses:   bs.text.TaskStatuses,
	}

	TextID := accs.GetTextIDfromURL(r.URL.Path)
	IntID, _ := strconv.Atoi(TextID)
	if TextID == "new" {
		Page.New = true
	} else {
		Page.Task = Task{ID: IntID}
		err = Page.Task.load(bs.db, bs.dbt)
		if err != nil {
			log.Println(accs.CurrentFunction()+":", err)
			http.NotFound(w, r)
			return
		}
	}

	user := unmarshalToProfile(bs.team.GetByID(Page.LoggedinID))
	Page.UserConfig = user.UserConfig
	if user.UserRole == ADMIN || Page.New || Page.Task.GiveCreatorID() == Page.LoggedinID {
		Page.Editable = true
	}
	Page.RemoveAllowed, _ = strconv.ParseBool(bs.cfg.RemoveAllowed)
	if user.UserRole == ADMIN {
		Page.RemoveAllowed = true
	}
	Page.IamAssignee = Page.Task.GiveAssigneeID() == Page.LoggedinID
	Page.IamParticipant = accs.SliceContainsInt(Page.Task.Participants, Page.LoggedinID)

	var created int
	var updated int

	var (
		eventAssigneeSet        = false
		eventTaskEdited         = false
		eventTaskStatusChanged  = false
		eventParticipantToAdded = false
		newParticipantID        = 0
		eventNewTaskComment     = false
		newCommentID            = 0
	)

	defaultUploadPath := filepath.Join(bs.cfg.ServerRoot, "files", "tasks", TextID)

	if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
		r.Body = http.MaxBytesReader(w, r.Body, MAX_UPLOAD_SIZE+1048576)
	}

	// Create or update code ==========================================================================
	if r.Method == "POST" && (r.FormValue("createButton") != "" || r.FormValue("updateButton") != "") {
		if Page.Editable == false {
			accs.ThrowAccessDenied(w, "writing task", Page.LoggedinID, IntID)
			return
		}
		t := Task{
			ID:         IntID,
			Created:    datetime.GetCurrentDateTime(),
			PlanStart:  datetime.StringToDateTime(r.FormValue("planStart")),
			PlanDue:    datetime.StringToDateTime(r.FormValue("planDue")),
			Creator:    &Profile{ID: Page.LoggedinID},
			Topic:      r.FormValue("topic"),
			Content:    r.FormValue("content"),
			TaskStatus: Page.Task.TaskStatus,
		}
		if r.FormValue("assignee") != "" && r.FormValue("assignee") != "0" {
			t.Assignee = &Profile{ID: accs.StrToInt(r.FormValue("assignee"))}
			if t.Assignee.ID != Page.Task.GiveAssigneeID() && Page.Task.TaskStatus < DONE {
				t.TaskStatus = ASSIGNED
				t.StatusSet = datetime.GetCurrentDateTime()
				eventAssigneeSet = true
			}
		} else {
			t.Assignee = nil
			if Page.Task.TaskStatus < DONE && Page.Task.TaskStatus != CREATED {
				t.TaskStatus = CREATED
				t.StatusSet = datetime.GetCurrentDateTime()
				eventTaskStatusChanged = true
			}
		}

		t.FileList, err = uploader(r, defaultUploadPath, "fileList")
		if err != nil {
			log.Println(accs.CurrentFunction()+":", err)
			Page.Message = "uploadError"
		}

		if r.FormValue("createButton") != "" && !strings.Contains(Page.Message, "Error") {
			t.StatusSet = datetime.GetCurrentDateTime()
			t.ID, created = t.create(bs.db, bs.dbt)
			if created > 0 {
				if eventAssigneeSet {
					t.Creator.preload(bs.db, bs.dbt)
					t.Assignee.preload(bs.db, bs.dbt)
					email := mail.EmailMessage{Subj: bs.i18n.Messages.Subj.AssigneeToSet + " [" + bs.i18n.TaskCaption + " #" + strconv.Itoa(t.ID) + "]"}
					if t.Assignee.Contacts.Email != "" && t.Assignee.UserLock == 0 {
						email.SendTo = append(email.SendTo, mail.UserToSend{Name: t.Assignee.FirstName + " " + t.Assignee.Surname, Email: t.Assignee.Contacts.Email})
					}
					taskMail := TaskMail{email.Subj, t, bs.i18n.Messages, bs.i18n.TaskCaption, bs.i18n.TaskStatuses, bs.systemURL}
					taskMail.constructToChannel(bs.db, bs.dbt, bs.taskmailtmpl, bs.mailchan, email, bs.regexes.emailCont)
				}
				moveUploadedFilesToFinalDest(defaultUploadPath,
					filepath.Join(bs.cfg.ServerRoot, "files", "tasks", strconv.Itoa(t.ID)),
					t.FileList)
				if Page.UserConfig.ReturnAfterCreation {
					redirAddr := "/tasks/?anyparticipants=my"
					if !Page.UserConfig.ShowFinishedTasks {
						redirAddr += "&taskstatuses=0&taskstatuses=1&taskstatuses=2&taskstatuses=3"
					}
					http.Redirect(w, r, redirAddr, http.StatusSeeOther)
				} else {
					http.Redirect(w, r, fmt.Sprintf("/tasks/task/%d", t.ID), http.StatusSeeOther)
				}
				return
			} else {
				Page.Message = "dataNotWritten"
			}
		}

		if r.FormValue("updateButton") != "" && !strings.Contains(Page.Message, "Error") {
			t.FileList = append(Page.Task.FileList, t.FileList...)
			updated = t.update(bs.db, bs.dbt)
			if updated > 0 {
				Page.Message = "dataWritten"
				Page.Task.load(bs.db, bs.dbt)
				eventTaskEdited = true
				eventTaskStatusChanged = false
			} else {
				Page.Message = "dataNotWritten"
				eventTaskEdited = false
				eventTaskStatusChanged = false
			}
		}

	}

	// Delete files ===========================================
	if r.Method == "POST" && r.FormValue("deleteFiles") != "" {
		if Page.Editable == false {
			accs.ThrowAccessDenied(w, "writing task", Page.LoggedinID, IntID)
			return
		}
		r.ParseForm()
		filesToRemove := r.Form["filesToRemove"]
		err = removeUploadedFiles(defaultUploadPath, filesToRemove)
		if err == nil {
			FileList := accs.FilterSliceStrList(Page.Task.FileList, filesToRemove)
			updated = sqla.UpdateSingleJSONListStr(bs.db, bs.dbt, "tasks", "FileList", FileList, IntID)
			if updated > 0 {
				Page.Message = "dataWritten"
				eventTaskEdited = true
				Page.Task.load(bs.db, bs.dbt)
			} else {
				Page.Message = "dataNotWritten"
			}
		} else {
			Page.Message = "removalError"
		}
	}

	// Forward task ========================================================================================
	if r.Method == "POST" && r.FormValue("assigneeForward") != "" && r.FormValue("assigneeForward") != "0" {
		if Page.Editable || Page.IamAssignee {
			oID := Page.Task.GiveAssigneeID()
			aID := accs.StrToInt(r.FormValue("assigneeForward"))
			if oID != aID {
				res := sqla.UpdateSingleInt(bs.db, bs.dbt, "tasks", "Assignee", aID, IntID)
				if res > 0 {
					Page.Task.Assignee = &Profile{ID: aID}
					Page.Task.Assignee.preload(bs.db, bs.dbt)
					Page.Message = "dataWritten"
					eventAssigneeSet = true
					if aID != oID && Page.Task.TaskStatus < DONE {
						t := Task{ID: IntID,
							StatusSet:  datetime.GetCurrentDateTime(),
							TaskStatus: ASSIGNED}
						ress := t.updateStatus(bs.db, bs.dbt)
						if ress > 0 {
							Page.Task.TaskStatus = ASSIGNED
							Page.Task.StatusSet = t.StatusSet
						}
					}
				} else {
					Page.Message = "dataNotWritten"
				}
			}
		} else {
			accs.ThrowAccessDenied(w, "forwarding task", Page.LoggedinID, IntID)
			return
		}
	}

	// Create Comments ==========================================
	if r.Method == "POST" && r.FormValue("createComment") != "" {
		if Page.Editable || Page.IamAssignee || Page.IamParticipant {
			c := Comment{
				Created: datetime.GetCurrentDateTime(),
				Creator: &Profile{ID: Page.LoggedinID},
				Task:    &Task{ID: IntID},
				Content: r.FormValue("content"),
			}
			var caff int
			c.ID, caff = c.create(bs.db, bs.dbt)
			if caff > 0 {
				Page.Message = "commentWritten"
				eventNewTaskComment = true
				newCommentID = c.ID
			} else {
				Page.Message = "commentNotWritten"
			}
			c.FileList, err = uploader(r, filepath.Join(defaultUploadPath, strconv.Itoa(c.ID)), "fileListComm")
			if err != nil {
				log.Println(accs.CurrentFunction()+":", err)
				Page.Message = "uploadError"
			} else {
				sqla.UpdateSingleJSONListStr(bs.db, bs.dbt, "comments", "FileList", c.FileList, c.ID)
			}
		} else {
			accs.ThrowAccessDenied(w, "writing comment", Page.LoggedinID, IntID)
			return
		}
	}

	// Add participant ===========================================
	if r.Method == "POST" && r.FormValue("participantAdd") != "" {
		if Page.Editable || Page.IamAssignee {
			pID := accs.StrToInt(r.FormValue("participantAdd"))
			if accs.SliceContainsInt(Page.Task.Participants, pID) {
				Page.Message = "participantAlreadyInList"
			} else {
				participants := append(Page.Task.Participants, pID)
				res := sqla.UpdateSingleJSONListInt(bs.db, bs.dbt, "tasks", "Participants", participants, IntID)
				if res > 0 {
					Page.Task.Participants = participants
					Page.Message = "dataWritten"
					eventParticipantToAdded = true
					newParticipantID = pID
				} else {
					Page.Message = "dataNotWritten"
				}
			}
		} else {
			accs.ThrowAccessDenied(w, "adding participant", Page.LoggedinID, IntID)
			return
		}
	}

	// Remove participant ===========================================
	if r.Method == "POST" && r.FormValue("participantRemove") != "" {
		if Page.Editable || Page.IamAssignee {
			pID := accs.StrToInt(r.FormValue("participantRemove"))
			participants := accs.FilterSliceInt(Page.Task.Participants, pID)
			res := sqla.UpdateSingleJSONListInt(bs.db, bs.dbt, "tasks", "participants", participants, IntID)
			if res > 0 {
				Page.Task.Participants = participants
				Page.Message = "dataWritten"
			} else {
				Page.Message = "dataNotWritten"
			}

		} else {
			accs.ThrowAccessDenied(w, "removing participant", Page.LoggedinID, IntID)
			return
		}
	}

	// Other fields code ============================================
	Page.UserList = bs.team.GetObjectArr("UserList")
	Page.IamAssignee = Page.Task.GiveAssigneeID() == Page.LoggedinID
	Page.IamParticipant = accs.SliceContainsInt(Page.Task.Participants, Page.LoggedinID)
	if TextID == "new" {
		Page.New = true
		Page.PageTitle = bs.text.NewTask
	} else {
		Page.PageTitle = bs.text.Task + " #" + TextID
		if Page.Task.Topic != "" {
			Page.PageTitle += ": " + Page.Task.Topic
		}
		//Status INPROGRESS on open
		if Page.IamAssignee && Page.Task.TaskStatus == ASSIGNED {
			t := Task{ID: IntID,
				StatusSet:  datetime.GetCurrentDateTime(),
				TaskStatus: INPROGRESS}
			res := t.updateStatus(bs.db, bs.dbt)
			if res > 0 {
				Page.Task.TaskStatus = INPROGRESS
				Page.Task.StatusSet = t.StatusSet
				eventTaskStatusChanged = true
			}
		}
		Page.Comments, err = Page.Task.loadComments(bs.db, bs.dbt)
		if err != nil {
			log.Println(accs.CurrentFunction()+":", err)
			accs.ThrowServerError(w, "loading task comments", Page.LoggedinID, Page.Task.ID)
			return
		}
		Page.Participants, err = Page.Task.loadParticipants(bs.db, bs.dbt)
		if err != nil {
			log.Println(accs.CurrentFunction()+":", err)
			accs.ThrowServerError(w, "loading task participants", Page.LoggedinID, Page.Task.ID)
			return
		}
		//Statuses all
		if Page.Editable || Page.IamAssignee {
			var statusCode int
			if r.Method == "POST" && r.FormValue("taskStatus") != "" {
				if r.FormValue("taskStatus") == "inprogress" {
					statusCode = INPROGRESS
				} else if r.FormValue("taskStatus") == "stuck" {
					statusCode = STUCK
				} else if r.FormValue("taskStatus") == "done" {
					statusCode = DONE
				} else if r.FormValue("taskStatus") == "cancelled" {
					statusCode = CANCELLED
				}
				if statusCode > 0 && statusCode < 6 {
					t := Task{ID: IntID,
						StatusSet:  datetime.GetCurrentDateTime(),
						TaskStatus: statusCode}
					res := t.updateStatus(bs.db, bs.dbt)
					if res > 0 {
						Page.Task.TaskStatus = statusCode
						Page.Task.StatusSet = t.StatusSet
						eventTaskStatusChanged = true
					}
				}
			}
		}
	}

	if DEBUG {
		evns := "Events:"
		if eventTaskEdited {
			evns += " eventTaskEdited"
		}
		if eventTaskStatusChanged {
			evns += " eventTaskStatusChanged"
		}
		if eventAssigneeSet {
			evns += " eventAssigneeSet"
		}
		if eventParticipantToAdded {
			evns += " eventParticipantToAdded"
		}
		if eventNewTaskComment {
			evns += " eventNewTaskComment"
		}
		log.Println(evns)
	}

	// Email messages ================================================
	if eventTaskEdited || eventTaskStatusChanged || eventAssigneeSet {
		email := mail.EmailMessage{}
		if eventTaskEdited {
			email.Subj = bs.i18n.Messages.Subj.TaskEdited + " [" + bs.i18n.TaskCaption + " #" + strconv.Itoa(Page.Task.ID) + "]"
		} else if eventAssigneeSet {
			email.Subj = bs.i18n.Messages.Subj.AssigneeSet + " [" + bs.i18n.TaskCaption + " #" + strconv.Itoa(Page.Task.ID) + "]"
		} else if eventTaskStatusChanged {
			email.Subj = bs.i18n.Messages.Subj.TaskStatusChanged + " [" + bs.i18n.TaskCaption + " #" + strconv.Itoa(Page.Task.ID) + "]"
		}
		if Page.Task.Creator != nil && Page.Task.Creator.Contacts.Email != "" && Page.Task.Creator.UserLock == 0 {
			email.SendTo = append(email.SendTo, mail.UserToSend{Name: Page.Task.Creator.FirstName + " " + Page.Task.Creator.Surname, Email: Page.Task.Creator.Contacts.Email})
		}
		if Page.Task.Assignee != nil && Page.Task.Assignee.Contacts.Email != "" && !eventAssigneeSet && Page.Task.Assignee.UserLock == 0 {
			email.SendTo = append(email.SendTo, mail.UserToSend{Name: Page.Task.Assignee.FirstName + " " + Page.Task.Assignee.Surname, Email: Page.Task.Assignee.Contacts.Email})
		}
		for i := 0; i < len(Page.Participants); i++ {
			if Page.Participants[i].Contacts.Email != "" && Page.Participants[i].UserLock == 0 {
				email.SendCc = append(email.SendCc, mail.UserToSend{Name: Page.Participants[i].FirstName + " " + Page.Participants[i].Surname, Email: Page.Participants[i].Contacts.Email})
			}
		}
		taskMail := TaskMail{email.Subj, Page.Task, bs.i18n.Messages, bs.i18n.TaskCaption, bs.i18n.TaskStatuses, bs.systemURL}
		taskMail.constructToChannel(bs.db, bs.dbt, bs.taskmailtmpl, bs.mailchan, email, bs.regexes.emailCont)
	}

	if eventAssigneeSet {
		email := mail.EmailMessage{Subj: bs.i18n.Messages.Subj.AssigneeToSet + " [" + bs.i18n.TaskCaption + " #" + strconv.Itoa(Page.Task.ID) + "]"}
		if Page.Task.Assignee != nil && Page.Task.Assignee.Contacts.Email != "" && Page.Task.Assignee.UserLock == 0 {
			email.SendTo = append(email.SendTo, mail.UserToSend{Name: Page.Task.Assignee.FirstName + " " + Page.Task.Assignee.Surname, Email: Page.Task.Assignee.Contacts.Email})
		}
		taskMail := TaskMail{email.Subj, Page.Task, bs.i18n.Messages, bs.i18n.TaskCaption, bs.i18n.TaskStatuses, bs.systemURL}
		taskMail.constructToChannel(bs.db, bs.dbt, bs.taskmailtmpl, bs.mailchan, email, bs.regexes.emailCont)
	}

	if eventParticipantToAdded {
		email := mail.EmailMessage{Subj: bs.i18n.Messages.Subj.ParticipantToAdded + " [" + bs.i18n.TaskCaption + " #" + strconv.Itoa(Page.Task.ID) + "]"}
		for i := 0; i < len(Page.Participants); i++ {
			if Page.Participants[i].ID == newParticipantID && Page.Participants[i].Contacts.Email != "" && Page.Participants[i].UserLock == 0 {
				email.SendTo = append(email.SendTo, mail.UserToSend{Name: Page.Participants[i].FirstName + " " + Page.Participants[i].Surname, Email: Page.Participants[i].Contacts.Email})
				break
			}
		}
		taskMail := TaskMail{email.Subj, Page.Task, bs.i18n.Messages, bs.i18n.TaskCaption, bs.i18n.TaskStatuses, bs.systemURL}
		taskMail.constructToChannel(bs.db, bs.dbt, bs.taskmailtmpl, bs.mailchan, email, bs.regexes.emailCont)
	}

	if eventNewTaskComment {
		email := mail.EmailMessage{Subj: bs.i18n.Messages.Subj.NewTaskComment + " [" + bs.i18n.TaskCaption + " #" + strconv.Itoa(Page.Task.ID) + "]"}
		if Page.Task.Creator != nil && Page.Task.Creator.Contacts.Email != "" && Page.Task.Creator.UserLock == 0 {
			email.SendTo = append(email.SendTo, mail.UserToSend{Name: Page.Task.Creator.FirstName + " " + Page.Task.Creator.Surname, Email: Page.Task.Creator.Contacts.Email})
		}
		if Page.Task.Assignee != nil && Page.Task.Assignee.Contacts.Email != "" && Page.Task.Assignee.UserLock == 0 {
			email.SendTo = append(email.SendTo, mail.UserToSend{Name: Page.Task.Assignee.FirstName + " " + Page.Task.Assignee.Surname, Email: Page.Task.Assignee.Contacts.Email})
		}
		for i := 0; i < len(Page.Participants); i++ {
			if Page.Participants[i].Contacts.Email != "" && Page.Participants[i].UserLock == 0 {
				email.SendCc = append(email.SendCc, mail.UserToSend{Name: Page.Participants[i].FirstName + " " + Page.Participants[i].Surname, Email: Page.Participants[i].Contacts.Email})
			}
		}
		if len(email.SendTo) > 0 || len(email.SendCc) > 0 {
			commMail := CommMail{email.Subj, Page.Task.ID, Page.Task.Topic, Comment{ID: 0}, 0, bs.i18n.Messages, bs.i18n.TaskCaption, bs.i18n.CommentCaption, bs.systemURL}
			for i := 0; i < len(Page.Comments); i++ {
				if Page.Comments[i].ID == newCommentID {
					commMail.Comment = Page.Comments[i]
					commMail.CommentIndex = i + 1
					break
				}
			}
			commMail.constructToChannel(bs.db, bs.dbt, bs.commmailtmpl, bs.mailchan, email, bs.regexes.emailCont)
		}
	}

	// JSON output
	if r.URL.Query().Get("api") == "json" || r.FormValue("api") == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Page)
		//jsonOut, _ := json.Marshal(Page)
		//fmt.Fprintln(w, string(jsonOut))
		return
	}

	// HTML output
	err = bs.templates.ExecuteTemplate(w, "task.tmpl", Page)
	if err != nil {
		log.Println(accs.CurrentFunction()+":", err)
		accs.ThrowServerError(w, "executing task template", Page.LoggedinID, Page.Task.ID)
		return
	}

}
