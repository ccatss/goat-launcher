package launcher

import (
	_ "embed"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ccatss/goat-launcher/auth"
	"golang.org/x/oauth2"
)

type Launcher struct {
	client          *auth.Client
	store           auth.Store
	currentSession  *auth.Session
	currentAccounts []auth.Account

	app    fyne.App
	window fyne.Window

	accountSelect   *widget.Select
	characterSelect *widget.Select

	selectedAccount   string
	selectedCharacter string

	// Path to RuneLite client jar
	runelitePath string
}

type Option func(l *Launcher)

func WithRuneLite(path string) Option {
	return func(l *Launcher) {
		l.runelitePath = path
	}
}

func New(opts ...Option) *Launcher {
	l := &Launcher{
		client:       &auth.Client{},
		store:        auth.NewStore(),
		runelitePath: "RuneLite.jar",
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

//go:embed Icon.png
var iconBytes []byte

func (l *Launcher) Start() {
	icon := fyne.NewStaticResource("Icon.png", iconBytes)

	l.app = app.New()
	l.app.SetIcon(icon)

	l.window = l.app.NewWindow("GoAT Launcher")
	l.window.Resize(fyne.NewSize(400, 200))

	l.window.SetIcon(icon)
	profiles := []string{"Add New Profile"}

	keys, err := l.store.List()

	if err == nil {
		profiles = append(keys, profiles...)
	}

	var characters []string

	l.characterSelect = widget.NewSelect(characters, func(value string) {
		l.selectedCharacter = value
	})
	l.characterSelect.PlaceHolder = "Select a character..."

	l.accountSelect = widget.NewSelect(profiles, func(value string) {
		l.selectedAccount = value

		if value == "Add New Profile" {
			// Prompt for new profile
			l.addProfile()
		} else {
			l.handleAccountSelect(value)
		}
	})

	if len(keys) > 0 {
		l.accountSelect.SetSelected(keys[0])
	}

	l.accountSelect.PlaceHolder = "Select a profile..."

	launchBtn := widget.NewButton("Launch", func() {
		if l.selectedAccount == "" || l.selectedCharacter == "" {
			l.showError("Please select an account and character.")
			return
		}

		l.launch()
	})

	launchBtn.Importance = widget.HighImportance

	removeProfileBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if l.selectedAccount == "" {
			return
		}

		confirmDialog := dialog.NewConfirm("Confirm Removal",
			fmt.Sprintf("Are you sure you want to remove the %s account?",
				l.selectedAccount),
			func(choice bool) {
				if !choice {
					return
				}

				err := l.store.Delete(l.selectedAccount)

				if err != nil {
					l.showError(fmt.Sprintf("Unable to remove account %s\n%v", l.selectedAccount, err))
				} else {
					l.refreshAccounts()
				}
			}, l.window)

		confirmDialog.SetConfirmText("Remove")
		confirmDialog.Show()
	})

	removeProfileBtn.Importance = widget.LowImportance

	accountCombo := container.NewHBox(
		container.NewStack(l.accountSelect),
		removeProfileBtn,
	)

	form := widget.NewForm(
		widget.NewFormItem("Account:", accountCombo),
		widget.NewFormItem("Character:", l.characterSelect),
	)

	content := container.NewVBox(
		form,
		layout.NewSpacer(),
		launchBtn,
	)

	// 6. Set content and run
	l.window.SetContent(content)
	l.window.ShowAndRun()
}

func (l *Launcher) showError(err string) {
	infoDialog := dialog.NewInformation("Error", err, l.window)

	infoDialog.Show()
}

func (l *Launcher) refreshAccounts() {
	keys, err := l.store.List()

	if err != nil {
		log.Fatal(err)
	}

	keys = append(keys, "Add New Profile")

	l.accountSelect.SetOptions(keys)

	if l.selectedAccount == "Add New Profile" {
		l.accountSelect.SetSelected(keys[0])
	}
}

func (l *Launcher) handleAccountSelect(value string) {
	session, err := l.store.Get(value)

	if err != nil {
		log.Fatal("Unable to get account", err)
	}

	l.currentSession = session

	accounts, err := l.client.Accounts(session)

	if err != nil {
		log.Fatal("Unable to retrieve characters", err)
	}

	l.currentAccounts = accounts

	if len(accounts) > 0 {
		accountNames := make([]string, len(accounts))

		for i, account := range accounts {
			accountNames[i] = account.DisplayName
		}

		l.characterSelect.SetOptions(accountNames)
		l.characterSelect.SetSelected(accountNames[0])
	} else {
		l.characterSelect.ClearSelected()
	}
}

func (l *Launcher) addProfile() {
	flow, err := auth.NewAuthFlow(OpenBrowser, func(id string, t *oauth2.Token) {
		info, err := l.client.UserInfo(t)

		if err != nil {
			l.showError(fmt.Sprintf("Unable to get user info %v", err))
			return
		}

		session, err := l.client.CreateSession(id)

		if err != nil {
			l.showError(fmt.Sprintf("Unable to create session: %v", err))
			return
		}

		if err := l.store.Set(info.Nickname, *session); err != nil {
			l.showError(fmt.Sprintf("Unable to save account to launcher settings\n%v", err))
			return
		}

		l.refreshAccounts()
	})

	if err != nil {
		log.Println("Error creating auth flow:", err)
		return
	}

	// Use our IPC Server to handle responses from the browser
	go func() {
		err := startIPCServer(flow)

		if err != nil {
			log.Println("Error starting IPC server:", err)
		}
	}()

	err = flow.Start()

	if err != nil {
		log.Println("Unable to start auth flow:", err)
		return
	}
}

func (l *Launcher) launch() {
	cmd := exec.Command("java", "-jar", l.runelitePath)

	var selectedAccount *auth.Account

	for _, account := range l.currentAccounts {
		fmt.Printf("Account %s, selected %s\n", account.DisplayName, l.selectedCharacter)

		if account.DisplayName == l.selectedCharacter {
			selectedAccount = &account
			break
		}
	}

	if selectedAccount == nil {
		l.showError(fmt.Sprintf("no account found for %s", l.selectedCharacter))
		return
	}

	env := NewEnv()

	env.Set("JX_SESSION_ID", l.currentSession.SessionID)
	env.Set("JX_CHARACTER_ID", selectedAccount.AccountID)
	env.Set("JX_DISPLAY_NAME", selectedAccount.DisplayName)

	cmd.Env = env.Slice()

	if err := cmd.Start(); err != nil {
		l.showError(fmt.Sprintf("Unable to launch RuneLite\n%v", err))
	}
}

// OpenBrowser opens the specified URL in the system's default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		// 'xdg-open' is the standard desktop utility on Linux for opening URLs/files
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		// 'open' is the native macOS command
		cmd = exec.Command("open", url)
	case "windows":
		// 'start' is a cmd built-in on Windows. We run it via cmd.exe
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Run the command and wait for it to execute
	return cmd.Start()
}

func parseJagexState(str string) url.Values {
	v := url.Values{}

	parts := strings.Split(str, ",")

	for _, part := range parts {
		idx := strings.Index(part, "=")

		if idx == -1 {
			break
		}

		key := part[0:strings.Index(part, "=")]
		value := part[idx+1:]

		v.Set(key, value)
	}

	return v
}
