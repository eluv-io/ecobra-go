package app

import (
	"github.com/spf13/cobra"
)

const (
	categoryKey = "category" // key for commands annotation
)

type CmdCategories []*CmdCategory

func (c CmdCategories) GetCategories() []*CmdCategory {
	return c
}

type CmdCategory struct {
	Name    string           `json:"name"`    // the key name of this category to be used in command
	Title   string           `json:"title"`   // title of the group that will be displayed in help
	Default bool             `json:"default"` // true to have the category be the default where commands with no category are
	Cmds    []*cobra.Command `json:"-"`
}

func NewCategories(categories []*CmdCategory, cmdRoot *cobra.Command) CmdCategories {
	return newCategoriesBuilder().
		with(categories).
		fillWith(cmdRoot.Commands()).
		build()
}

func annotateCmdCategory(c *cobra.Command, category string) {
	if c.Annotations == nil {
		c.Annotations = make(map[string]string)
	}
	c.Annotations[categoryKey] = category
}

// ----- builder -----
type categoriesBuilder struct {
	groups []*CmdCategory
	others *CmdCategory
}

func ng() []*cobra.Command {
	return make([]*cobra.Command, 0)
}

func newCategoriesBuilder() *categoriesBuilder {
	return &categoriesBuilder{}
}

func (cg *categoriesBuilder) with(appCategories []*CmdCategory) *categoriesBuilder {
	var others *CmdCategory
	for _, c := range appCategories {
		if c.Default && others == nil {
			others = c
		}
		c.Cmds = ng()
	}
	cg.others = others
	cg.groups = appCategories
	return cg
}

func (cg *categoriesBuilder) addCommand(c *cobra.Command) {
	added := false

	groupName := c.Annotations[categoryKey]
	if groupName == "" && c.Name() == "help" {
		// we don't add 'help' by ourselves but we want it in the base group
		groupName = cg.groups[0].Name
	}

	for _, g := range cg.groups {
		if groupName == g.Name {
			g.Cmds = append(g.Cmds, c)
			added = true
		}
	}
	if !added {
		cg.others.Cmds = append(cg.others.Cmds, c)
	}
}

func (cg *categoriesBuilder) fillWith(cmds []*cobra.Command) *categoriesBuilder {
	for _, c := range cmds {
		if c.Hidden {
			continue
		}
		cg.addCommand(c)
	}
	return cg
}

func (cg *categoriesBuilder) build() []*CmdCategory {
	return cg.groups
}
