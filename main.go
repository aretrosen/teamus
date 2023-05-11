package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dhowden/tag"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

type track struct {
	title       string
	description string
	idx         int
}

func (t track) Title() string       { return t.title }
func (t track) FilterValue() string { return t.title }
func (t track) Description() string { return t.description }

type audiofile struct {
	file        *os.File
	filename    string
	audioformat musicType
}

type audioKeyMap struct {
	Play         key.Binding
	TogglePause  key.Binding
	ToggleRepeat key.Binding
	SeekRight    key.Binding
	SeekLeft     key.Binding
	Delete       key.Binding
	MoveUp       key.Binding
	MoveDown     key.Binding
	VolumeUp     key.Binding
	VolumeDown   key.Binding
}

type model struct {
	list             list.Model
	err              error
	keys             *audioKeyMap
	audioContext     *audio.Context
	progress         progress.Model
	progressStr      string
	currentlyPlaying track
	rewind           bool
}

type tickMsg time.Time

var (
	player     *Player
	fileList   []audiofile
	playstatus string
)

func newAudioKeyMap() *audioKeyMap {
	return &audioKeyMap{
		Play: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select & play"),
		),
		TogglePause: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle pause"),
		),
		ToggleRepeat: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "toggle repeat"),
		),
		SeekRight: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "seek right"),
		),
		SeekLeft: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "seek left"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete"),
		),
		MoveUp: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "move up"),
		),
		MoveDown: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "move later"),
		),
		VolumeUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "increase volume"),
		),
		VolumeDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "decrease volume"),
		),
	}
}

func newModel(trackList []list.Item) *model {
	listKeys := newAudioKeyMap()
	newmodel := model{keys: listKeys}

	newmodel.list = list.New(trackList, list.NewDefaultDelegate(), 0, 0)
	newmodel.list.Title = "Tracks"

	// I needed to change default behavior as otherwise we would need to use
	// unintuitive keybindings
	newmodel.list.KeyMap.CursorDown = key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "down"),
	)
	newmodel.list.KeyMap.CursorUp = key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "up"),
	)
	newmodel.list.KeyMap.PrevPage = key.NewBinding(
		key.WithKeys("h", "pgup"),
		key.WithHelp("h/pgup", "prev page"),
	)
	newmodel.list.KeyMap.NextPage = key.NewBinding(
		key.WithKeys("l", "pgdown"),
		key.WithHelp("h/pgdn", "next page"),
	)
	newmodel.list.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.Play,
			listKeys.TogglePause,
			listKeys.ToggleRepeat,
			listKeys.SeekRight,
			listKeys.SeekLeft,
			listKeys.Delete,
			listKeys.MoveUp,
			listKeys.MoveDown,
			listKeys.VolumeUp,
			listKeys.VolumeDown,
		}
	}

	newmodel.progress = progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage())
	playstatus = "  "

	newmodel.audioContext = audio.NewContext(sampleRate)
	return &newmodel
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 2 - len(playstatus)
		m.list.SetSize(msg.Width, msg.Height-2)
		return m, nil

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch {
		case key.Matches(msg, list.DefaultKeyMap().Quit):
			if player != nil {
				player.Close()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keys.Play):
			m.currentlyPlaying = m.list.SelectedItem().(track)
			idx := m.currentlyPlaying.idx

			if player != nil {
				player.Rewind(idx)
			}

			var err error
			player, err = NewPlayer(fileList[idx].file, m.audioContext, fileList[idx].audioformat)
			if err != nil {
				m.list.NewStatusMessage("Cannot Play Audio: " + err.Error())
				m.progressStr = "--/--"
			} else {
				playstatus = "  "
				m.list.NewStatusMessage("Current Song: " + m.currentlyPlaying.title)
			}

			return m, nil

		case key.Matches(msg, m.keys.VolumeUp):
			if player != nil {
				player.InreaseVolume()
			}
			return m, nil

		case key.Matches(msg, m.keys.VolumeDown):
			if player != nil {
				player.DecreaseVolume()
			}
			return m, nil

		case key.Matches(msg, m.keys.SeekRight):
			if player != nil {
				player.SeekRight()
			}
			return m, nil

		case key.Matches(msg, m.keys.SeekLeft):
			if player != nil {
				player.SeekLeft()
			}
			return m, nil

		case key.Matches(msg, m.keys.TogglePause):
			if player != nil {
				playstatus = player.TogglePause()
			}
			return m, nil

			// TODO: Rewind still not implemented.
		case key.Matches(msg, m.keys.ToggleRepeat):
			m.rewind = !m.rewind
			return m, nil
		}

	case tickMsg:
		fracDone := 0.0
		if player != nil {
			m.progressStr, fracDone = player.Update()
		}
		cmd := m.progress.SetPercent(fracDone)
		return m, tea.Batch(tickCmd(), cmd)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) View() string {
	m.progress.Width = m.progress.Width - len(m.progressStr)
	progressive := " " + m.progress.View() + playstatus + m.progressStr + " \n"
	return lipgloss.JoinVertical(lipgloss.Left, progressive, m.list.View())
}

func listPaths(roots []string) {
	for _, root := range roots {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			for _, ext := range []string{".mp3", ".wav", ".oga", ".ogg", ".spx", ".opus", ".flac"} {
				if filepath.Ext(d.Name()) == ext {
					fileList = append(fileList, audiofile{
						filename: path,
					})
					break
				}
			}
			return nil
		})
	}
}

func main() {
	// TODO: Store it in a database. Searching for files everytime is just plain
	// retarded. Also add ability to add more paths via json.
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Cannot find Home Directory" + err.Error())
	}
	musdir := filepath.Join(home, "Music")
	if _, err := os.Stat(musdir); err != nil {
		log.Fatal("Could not find or open Music Directory" + err.Error())
	}

	listPaths([]string{musdir})
	var trackList []list.Item

	noOfFiles := len(fileList)

	for i := 0; i < noOfFiles; i++ {
		file, err := os.Open(fileList[i].filename)
		if err != nil {
			log.Fatal(err)
		}

		tr, err := tag.ReadFrom(file)
		if err != nil {
			file.Close()
			continue
		}

		var audioformat musicType
		switch tr.FileType() {
		case "MP3":
			audioformat = typeMP3
		case "OGG":
			audioformat = typeOgg
		case "FLAC":
			audioformat = typeFlac
		default:
			audioformat = typeWav
		}

		fileList[i].file, fileList[i].audioformat = file, audioformat
		trackList = append(trackList, track{
			idx:         i,
			title:       tr.Title(),
			description: tr.Album() + "⋅" + tr.Artist(),
		})
	}

	m := newModel(trackList)
	t := tea.NewProgram(*m, tea.WithAltScreen())

	if _, err := t.Run(); err != nil {
		log.Fatal(err)
	}

	if player != nil {
		player.Close()
	}

	for i := 0; i < noOfFiles; i++ {
		fileList[i].file.Close()
	}
}
