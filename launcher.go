package launcher

import (
	_ "embed"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path"
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
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type Launcher struct {
	client          *auth.Client
	store           auth.Store
	currentSession  *auth.Session
	currentAccounts []auth.Account

	app    fyne.App
	icon   fyne.Resource
	window fyne.Window

	accountSelect   *widget.Select
	characterSelect *widget.Select

	selectedAccount   string
	selectedCharacter string

	config        *viper.Viper
	dataDirectory string
	promptForCode bool
}

type Option func(l *Launcher)

func WithConfig(v *viper.Viper) Option {
	return func(l *Launcher) {
		l.config = v
	}
}

func WithDataDirectory(path string) Option {
	return func(l *Launcher) {
		l.dataDirectory = path
	}
}

func WithStore(store auth.Store) Option {
	return func(l *Launcher) {
		l.store = store
	}
}

func PromptForCode() Option {
	return func(l *Launcher) {
		l.promptForCode = true
	}
}

func New(opts ...Option) (*Launcher, error) {
	icon := fyne.NewStaticResource("Icon.png", iconBytes)

	l := &Launcher{
		client: &auth.Client{},
		app:    app.New(),
		icon:   icon,
		config: viper.GetViper(),
	}

	for _, opt := range opts {
		opt(l)
	}

	if l.store == nil {
		store, err := auth.NewStore()

		if err != nil {
			return nil, err
		}

		l.store = store
	}

	return l, nil
}

//go:embed cmd/Icon.png
var iconBytes []byte

func (l *Launcher) Start() {
	l.window = l.app.NewWindow("GoAT Launcher")
	l.window.Resize(fyne.NewSize(400, 200))
	l.window.SetIcon(l.icon)

	var characters []string

	l.characterSelect = widget.NewSelect(characters, func(value string) {
		l.selectedCharacter = value
	})

	l.characterSelect.PlaceHolder = "Select a character..."

	l.accountSelect = widget.NewSelect([]string{"No Account", "Add New Account"}, func(value string) {
		l.selectedAccount = value

		if value == "Add New Account" {
			// Prompt for new profile
			go l.addProfile()

			l.accountSelect.ClearSelected()
		} else if value == "No Account" {
			// Disable character select
			l.characterSelect.Disable()
		} else if value != "" {
			if l.characterSelect.Disabled() {
				l.characterSelect.Enable()
			}

			go l.handleAccountSelect(value)
		}
	})

	l.accountSelect.PlaceHolder = "Select a profile..."

	launchBtn := widget.NewButton("Launch", func() {
		if l.selectedAccount == "" || l.selectedAccount != "No Account" && l.selectedCharacter == "" {
			l.showError("Please select an account and character.")
			return
		}

		l.launch()
	})

	launchBtn.Importance = widget.HighImportance

	removeAccountBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
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

	removeAccountBtn.Importance = widget.LowImportance

	accountCombo := container.NewHBox(
		container.NewStack(l.accountSelect),
		removeAccountBtn,
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

	l.window.SetContent(content)

	go l.populateData()

	l.window.ShowAndRun()
}

// populateData is called after the window is visible so that we can show dialogs first
func (l *Launcher) populateData() {
	keys, err := l.store.List()

	profiles := []string{"No Account", "Add New Account"}

	if err == nil {
		profiles = append(keys, profiles...)
	} else {
		fmt.Println("Error reading keys from store:", err)
	}

	fyne.Do(func() {
		l.accountSelect.SetOptions(profiles)

		if len(keys) > 0 {
			l.accountSelect.SetSelected(keys[0])
		}
	})
}

func (l *Launcher) showDialog(title, message string) {
	popup := l.app.NewWindow(title)
	popup.Resize(fyne.NewSize(400, 200))
	popup.SetFixedSize(true)

	popup.CenterOnScreen()

	errorIcon := widget.NewIcon(theme.ErrorIcon())

	msgLabel := widget.NewLabel(message)
	msgLabel.Wrapping = fyne.TextWrapWord

	bodyLayout := container.NewBorder(nil, nil, errorIcon, nil, msgLabel)

	dismissBtn := widget.NewButton("Dismiss", func() {
		popup.Close()
	})
	dismissBtn.Importance = widget.HighImportance

	content := container.NewPadded(
		container.NewVBox(
			bodyLayout,
			dismissBtn,
		),
	)

	popup.SetContent(content)

	popup.Show()
}

func (l *Launcher) showError(err string) {
	fyne.Do(func() {
		l.showDialog("Error", err)
	})
}

type PromptOpts struct {
	Title       string
	Prompt      string
	InputType   string
	PlaceHolder string
	Callback    func(string)
}

func (l *Launcher) Prompt(opts PromptOpts) {
	popup := l.app.NewWindow(opts.Title)
	popup.Resize(fyne.NewSize(420, 180))

	popup.CenterOnScreen()

	promptLabel := widget.NewLabel(opts.Prompt)
	promptLabel.Wrapping = fyne.TextWrapWord

	var entry *widget.Entry

	switch opts.InputType {
	case "password":
		entry = widget.NewPasswordEntry()
	default:
		entry = widget.NewEntry()
	}
	entry.PlaceHolder = opts.PlaceHolder

	submitBtn := widget.NewButton("Submit", func() {
		opts.Callback(entry.Text)
		popup.Close()
	})
	submitBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButton("Cancel", func() {
		popup.Close()
	})
	cancelBtn.Importance = widget.HighImportance

	layoutStack := container.NewVBox(
		promptLabel,
		entry,
		layout.NewSpacer(),
		container.NewGridWithColumns(2, submitBtn, cancelBtn),
	)

	// Wrap in a padded container so elements don't hug the absolute edges of the modal frame
	dialogContent := container.NewPadded(layoutStack)

	submitBtn.OnTapped = func() {
		opts.Callback(entry.Text)
		popup.Close()
	}

	popup.SetContent(dialogContent)

	popup.Show()
	popup.Canvas().Focus(entry)
}

func (l *Launcher) refreshAccounts() {
	keys, err := l.store.List()

	if err != nil {
		log.Fatal(err)
	}

	keys = append(keys, "Add New Account")

	fyne.Do(func() {
		l.accountSelect.SetOptions(keys)

		if l.selectedAccount == "Add New Account" {
			l.accountSelect.SetSelected(keys[0])
		}
	})
}

func (l *Launcher) handleAccountSelect(value string) {
	session, err := l.store.Get(value)

	if err != nil {
		slog.Error("Unable to retrieve session from store", slog.Any("error", err))
		l.showError(fmt.Sprintf("Unable to retrieve account %s\n%v", value, err))
		return
	}

	l.currentSession = session

	accounts, err := l.client.Accounts(session)

	if err != nil {
		slog.Error("Unable to retrieve characters!", slog.Any("error", err))
		l.showError(fmt.Sprintf("Unable to retrieve characters due to error\n%v", err))
		return
	}

	l.currentAccounts = accounts

	fyne.Do(func() {
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
	})
}

func (l *Launcher) addProfile() {
	openUrl := func(dest string) error {
		u, err := url.Parse(dest)

		if err != nil {
			slog.Error("Unable to parse URL", slog.Any("error", err))
			return err
		}

		return l.app.OpenURL(u)
	}

	flow, err := auth.NewAuthFlow(openUrl, func(id string, t *oauth2.Token) {
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
			l.showError(fmt.Sprintf("Unable to save account to launcher store\n%v", err))
			return
		}

		l.refreshAccounts()
	})

	if err != nil {
		slog.Error("Error creating auth flow", slog.Any("error", err))
		return
	}

	if l.promptForCode {
		fyne.Do(func() {
			l.Prompt(PromptOpts{
				Title:     "Enter the Launcher URL",
				Prompt:    "Once logged in, if prompted click Cancel on the popup and copy the `Return To Launcher` link here.",
				InputType: "text",
				Callback: func(str string) {
					v := parseJagexState(str)

					err := flow.Exchange(v.Get("code"), v.Get("state"))

					if err != nil {
						l.showError(fmt.Sprintf("Unable to exchange code %v", err))
						return
					}
				},
			})
		})
	} else {
		// Use our IPC Server to handle responses from the browser
		go func() {
			err := startIPCServer(flow)

			if err != nil {
				slog.Error("Unable to start IPC Server", slog.Any("error", err))
			}
		}()
	}

	err = flow.Start()

	if err != nil {
		slog.Error("Unable to start auth flow", slog.Any("error", err))
		return
	}
}

func (l *Launcher) launch() {
	runelitePath := l.config.GetString("runelite-path")

	if runelitePath == "" {
		clientPath := path.Join(l.dataDirectory, "RuneLite.jar")

		if _, err := os.Stat(clientPath); os.IsNotExist(err) {
			// Download RuneLite to dataPath
			err = DownloadRuneLite(clientPath)

			if err != nil {
				slog.Error("Unable to download RuneLite", slog.String("error", err.Error()))
				l.showError("RuneLite Launching Failed.\nClient does not exist and downloading from github failed.")
			}
		}

		runelitePath = clientPath
	}

	cmd := exec.Command("java", "-jar", runelitePath)

	var selectedAccount *auth.Account

	if l.selectedAccount != "No Account" {
		for _, account := range l.currentAccounts {
			if account.DisplayName == l.selectedCharacter {
				selectedAccount = &account
				break
			}
		}

		if selectedAccount == nil {
			l.showError(fmt.Sprintf("no account found for %s", l.selectedCharacter))
			return
		}
	}

	env := NewEnv()

	if selectedAccount != nil {
		env.Set("JX_SESSION_ID", l.currentSession.SessionID)
		env.Set("JX_CHARACTER_ID", selectedAccount.AccountID)
		env.Set("JX_DISPLAY_NAME", selectedAccount.DisplayName)
	}

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
