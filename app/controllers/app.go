package controllers

import (
    "github.com/robfig/revel"
    "bnginx/app/models"
	"bnginx/app/routes"
)


type App struct {
    GorpController
}

// ================ Helper functions ================

func (c App) addPages() revel.Result {
	var pages []*models.Post
	_, err := c.Txn.Select(&pages, `SELECT * FROM Post WHERE Published AND IsPage ORDER BY PageOrder ASC`)
	if err != nil {
		revel.ERROR.Panic(err)
	}
	c.RenderArgs["pages"] = pages
	return nil
}

func (c App) addUser() revel.Result {
    if user := c.connected(); user != nil {
         c.RenderArgs["user"] = user
    }
    return nil
}

func (c App) connected() *models.User {
    if c.RenderArgs["user"] != nil {
        return c.RenderArgs["user"].(*models.User)
    }
    if email, ok := c.Session["user"]; ok {
        return c.getUser(email)
    }
    return nil
}

func (c App) getUser(email string) *models.User {
    users, err := c.Txn.Select(models.User{}, `SELECT * FROM User WHERE Email = ?`, email)
    if err != nil {
		revel.ERROR.Panic(err)
		return nil
    }
    if len(users) == 0 {
		c.Flash.Error("No result for this email")
        return nil
    }
    return users[0].(*models.User)
}

func (c App) getUserById(userId int64) *models.User {
	obj, err := c.Txn.Get(models.User{}, userId)
	if err != nil {
		revel.ERROR.Panic(err)
		return nil
    } else if obj == nil {
		c.Flash.Error("No result for this id")
		return nil
	}
	return obj.(*models.User)
}

func (c App) getPostById(postId int64) *models.Post {
	obj, err := c.Txn.Get(models.Post{}, postId)
	if err != nil {
		revel.ERROR.Panic(err)
	} else if obj == nil {
		c.Flash.Error("No result for this id")
		return nil
	}
	return obj.(*models.Post)
}

func (c App) getPublishedPosts(offset int64) []*models.Post {
	var posts []*models.Post
	_, err := c.Txn.Select(&posts, `SELECT * FROM Post WHERE Published ORDER BY Updated DESC LIMIT 5 OFFSET ?`, offset)
	if err != nil {
		revel.ERROR.Panic(err)
	}
	return posts
}

// ================ Actions ================

func (c App) Login(email string, password string) revel.Result {
	c.Validation.Required(email)
	c.Validation.Required(password)
	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(routes.App.Index(0))
	}
	
    user := c.getUser(email)
    if user == nil || !user.CheckPassword(password) {
        c.Flash.Error("Wrong email or password")
        return c.Redirect(routes.App.Index(0))
    }
    c.Session["user"] = user.Email
    c.RenderArgs["User"] = user // Probably not needed
    return c.Redirect(routes.Admin.Index())
}

func (c App) Logout() revel.Result {
	delete(c.Session, "user")
	delete(c.RenderArgs, "user")
	return c.Redirect(routes.App.Index(0))
}

func (c App) Index(offset int64) revel.Result {
	posts := c.getPublishedPosts(offset)
	return c.Render(posts)
}

func (c App) Search(query string, offset int64) revel.Result {
	var posts []*models.Post
	q := "%"+query+"%"
	
	sql := `SELECT * FROM Post WHERE Published AND (Body like ? OR Title like ?) LIMIT 10 OFFSET ?`
	_, err := c.Txn.Select(&posts, sql, q, q, offset)
	if err != nil {
		revel.ERROR.Panic(err.Error())
	}
	return c.Render(posts, query)
}

func (c App) Post(postId int64) revel.Result {
	post := c.getPostById(postId)
	//user := c.connected()
	
	// TODO Maybe a hidden property would be better
	if post == nil {//|| !page.Published && (user == nil || !user.IsAdmin) {
		return c.Redirect(routes.App.Index(0))
	}
	return c.Render(post)
}
