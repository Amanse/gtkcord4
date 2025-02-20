package guilds

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/diamondburned/ningen/v3"
)

const (
	FolderSize     = 32
	FolderMiniSize = 16
)

// Folder is the widget containing the folder icon on top and a child list of
// guilds beneath it.
type Folder struct {
	*gtk.Box

	ButtonOverlay *gtk.Overlay
	ButtonPill    *Pill
	Button        *FolderButton

	Name *NamePopover

	Revealer *gtk.Revealer
	GuildBox *gtk.Box
	Guilds   []*Guild

	ctx  context.Context
	ctrl GuildOpener
	open bool

	// TODO: if we ever track the unread indicator, then our unread container
	// will need to count the guilds for us for SetGuildOpen to work
	// correctly.
}

var folderCSS = cssutil.Applier("guild-folder", `
	.guild-folder .guild-guild > button {
		padding: 0 12px;
	}
	.guild-folder .guild-guild > button > .adaptive-avatar {
		padding: 4px 0;
		transition: 200ms ease;
		background-color: @theme_bg_color;
	}
	.guild-folder .guild-guild:last-child > button {
		padding-bottom: 4px;
	}
	.guild-folder .guild-guild:last-child > button > .adaptive-avatar {
		padding: 0;
		padding-top: 4px;
		border-radius: 0 0 99px 99px;
		background-color: @theme_bg_color;
	}
	.guild-folder .guild-guild > button:hover > .adaptive-avatar {
		background-color: mix(@borders, @theme_bg_color, 0.5);
	}
`)

// NewFolder creates a new Folder.
func NewFolder(ctx context.Context, ctrl GuildOpener) *Folder {
	f := Folder{
		ctx:  ctx,
		ctrl: ctrl,
	}

	f.Button = NewFolderButton(ctx)
	f.Button.SetRevealed(false)
	f.Button.ConnectClicked(f.toggle)

	f.ButtonPill = NewPill()

	f.ButtonOverlay = gtk.NewOverlay()
	f.ButtonOverlay.SetChild(f.Button)
	f.ButtonOverlay.AddOverlay(f.ButtonPill)

	f.Name = NewNamePopover()
	f.Name.Bind(f.Button)

	f.GuildBox = gtk.NewBox(gtk.OrientationVertical, 0)

	f.Revealer = gtk.NewRevealer()
	f.Revealer.SetTransitionType(gtk.RevealerTransitionTypeSlideDown)
	f.Revealer.SetRevealChild(false)
	f.Revealer.SetChild(f.GuildBox)

	f.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	f.Box.Append(f.ButtonOverlay)
	f.Box.Append(f.Revealer)
	f.AddCSSClass("guild-folder-collapsed")
	folderCSS(f.Box)

	return &f
}

// Unselect unselects the folder visually.
func (f *Folder) Unselect() {
	f.setGuildOpen(false)
}

func (f *Folder) setGuildOpen(open bool) {
	f.open = open

	if f.Revealer.RevealChild() {
		f.ButtonPill.State = PillOpened
	} else {
		if open {
			f.ButtonPill.State = PillActive
		} else {
			f.ButtonPill.State = PillInactive
		}
	}

	f.ButtonPill.Invalidate()
}

func (f *Folder) toggle() {
	reveal := !f.Revealer.RevealChild()
	f.Revealer.SetRevealChild(reveal)
	f.Button.SetRevealed(reveal)
	f.setGuildOpen(f.open)

	if reveal {
		f.RemoveCSSClass("guild-folder-collapsed")
	} else {
		f.AddCSSClass("guild-folder-collapsed")
	}
}

// Set sets a fresh list of guilds.
func (f *Folder) Set(folder *gateway.GuildFolder) {
	f.Name.SetName(folder.Name)
	f.Button.SetIcons(folder.GuildIDs)

	if folder.Color != discord.NullColor {
		f.Button.SetColor(folder.Color)
	}

	for _, guild := range f.Guilds {
		f.GuildBox.Remove(guild)
	}

	f.Guilds = make([]*Guild, len(folder.GuildIDs))

	for i, id := range folder.GuildIDs {
		g := NewGuild(f.ctx, (*guildOpenerFolder)(f), id)
		g.SetParentFolder(f)

		f.Guilds[i] = g
		f.GuildBox.Append(g)
	}

	// Invalidate after, since guilds can call back onto our Folder list.
	for _, g := range f.Guilds {
		g.Invalidate()
	}
}

// Remove removes the given guild by its ID.
func (f *Folder) Remove(id discord.GuildID) {
	for i, guild := range f.Guilds {
		if guild.ID() == id {
			f.GuildBox.Remove(guild)
			f.Guilds = append(f.Guilds[:i], f.Guilds[i+1:]...)
			return
		}
	}
}

func (f *Folder) viewChild() {}

// InvalidateUnread invalidates the folder's unread state.
func (f *Folder) InvalidateUnread() {
	var unread ningen.UnreadIndication
	for _, guild := range f.Guilds {
		if guild.unread > unread {
			unread = guild.unread
		}
	}

	switch unread {
	case ningen.ChannelRead:
		f.ButtonPill.Attrs = 0
	case ningen.ChannelUnread:
		f.ButtonPill.Attrs = PillUnread
	case ningen.ChannelMentioned:
		f.ButtonPill.Attrs = PillMentioned
	}

	f.ButtonPill.Invalidate()
}

type guildOpenerFolder Folder

func (f *guildOpenerFolder) OpenGuild(id discord.GuildID) {
	(*Folder)(f).setGuildOpen(true)
	f.ctrl.OpenGuild(id)
}
