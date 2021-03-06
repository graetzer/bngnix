package controllers

import (
	"bnginx/app/models"
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/graetzer/go-recaptcha"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"

	// importing database driver
	"github.com/jinzhu/gorm"
	// import dialect
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/revel/revel"
)

func init() {
	revel.OnAppStart(AppInit)
	revel.InterceptMethod((*Base).addTemplateVars, revel.BEFORE)
	revel.InterceptMethod((*Admin).checkUser, revel.BEFORE)
	revel.InterceptMethod((*App).addCacheHeaders, revel.AFTER)

	revel.TemplateFuncs["markdown"] = func(input string) template.HTML {
		return template.HTML(string(blackfriday.MarkdownCommon([]byte(input))))
	}
	// Use for user supplied content, sanitizes html input and javascript
	revel.TemplateFuncs["markdownSave"] = func(input string) template.HTML {
		return template.HTML(string(secureMarkdown([]byte(input))))
	}

	revel.TemplateFuncs["sub"] = func(a, b int) int {
		return a - b
	}

	revel.TemplateFuncs["add"] = func(a, b int) int {
		return a + b
	}

	revel.TemplateFuncs["username"] = func(userID int64) string {
		var user models.User
		if DB.First(&user, userID).RecordNotFound() {
			return ""
		}
		return user.Name
	}

	revel.TemplateFuncs["commentCount"] = func(post *models.Blogpost) int64 {
		var result int64
		if post == nil {
			DB.Model(models.Comment{}).Where("NOT approved").Count(&result)
		} else {
			DB.Model(models.Comment{}).Where("post_id = ? AND approved", post.ID).Count(&result)
		}
		return result
	}

	revel.TemplateFuncs["place"] = func(stay models.Stay) *models.Place {
		var place models.Place
		if DB.First(&place, stay.PlaceID).RecordNotFound() {
			return nil
		}
		return &place
	}

	revel.TemplateFuncs["json"] = func(v interface{}) string {
		if v == nil {
			return "(nil)"
		}
		bytes, err := json.Marshal(v)
		if err != nil {
			return "" //fmt.Println("error:", err)
		}
		return string(bytes)
	}

	revel.TemplateFuncs["eqtime"] = func(t1 time.Time, t2 time.Time) bool {
		return t1.Equal(t2)
	}
	// not pretty but effective
	patterns := [...]string{"0.png", "1.gif", "2.jpg", "3.png", "4.png",
		"5.png", "6.jpg", "7.png", "8.png", "9.gif"}
	revel.TemplateFuncs["pattern"] = func(id int64) string {
		i := int(id) % len(patterns)
		return "/public/patterns/pattern" + patterns[i]
	}
}

var (
	// global DB instance
	DB     *gorm.DB
	policy *bluemonday.Policy
)

func AppInit() {
	// Init HTML sanitizer
	policy = bluemonday.UGCPolicy()

	os.MkdirAll(DataBaseDir(), 0777)
	db, err := gorm.Open("sqlite3", filepath.Join(DataBaseDir(), "sqlite_bnginx.db"))
	if err == nil {
		db.LogMode(revel.DevMode)
		if !db.HasTable(&models.User{}) {
			db.CreateTable(&models.User{})
			db.CreateTable(&models.Blogpost{})
			db.CreateTable(&models.Comment{})
			db.CreateTable(&models.Project{})
			db.CreateTable(&models.Place{})
			db.CreateTable(&models.Stay{})
		}

		var user models.User // Add default admin user
		if db.First(&user).RecordNotFound() {
			user = models.User{Name: "Simon", Email: "simon@graetzer.org", IsAdmin: true}
			user.SetPassword("default")
			db.Create(&user)
		}
	}
	DB = db

	if secret, found := revel.Config.String("recaptcha.secret"); found {
		recaptcha.Init(secret)
	}
}

func secureMarkdown(input []byte) []byte {
	// set up the HTML renderer
	htmlFlags := 0
	htmlFlags |= blackfriday.HTML_USE_XHTML
	htmlFlags |= blackfriday.HTML_USE_SMARTYPANTS
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_FRACTIONS
	htmlFlags |= blackfriday.HTML_SKIP_HTML
	htmlFlags |= blackfriday.HTML_SKIP_STYLE
	htmlFlags |= blackfriday.HTML_SKIP_IMAGES
	htmlFlags |= blackfriday.HTML_SAFELINK
	renderer := blackfriday.HtmlRenderer(htmlFlags, "", "")

	// set up the parser
	extensions := 0
	extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	extensions |= blackfriday.EXTENSION_TABLES
	extensions |= blackfriday.EXTENSION_FENCED_CODE
	extensions |= blackfriday.EXTENSION_AUTOLINK
	extensions |= blackfriday.EXTENSION_STRIKETHROUGH
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS

	unsafe := blackfriday.Markdown(input, renderer, extensions)
	return policy.SanitizeBytes(unsafe)
}

// DataBaseDir returns the path where to store locale data
func DataBaseDir() string {
	base, found := revel.Config.String("databasedir")
	if !found {
		if runtime.GOOS == "windows" {
			base = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
			if base == "" {
				base = os.Getenv("USERPROFILE")
			}
		} else {
			base = os.Getenv("HOME")
		}
		base = filepath.Join(base, "/bnginx-data")
	}
	//revel.INFO.Println("Using basepath " + base)
	return base
}
