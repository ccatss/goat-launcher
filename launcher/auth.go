package launcher

import (
	"context"
	"net/http"
	"time"

	"github.com/ccatss/goat-launcher/auth"
	ipc "github.com/james-barrow/golang-ipc"
)

func startIPCServer(flow *auth.Flow) error {
	s, err := ipc.StartServer("goat", nil)

	if err != nil {
		return err
	}

	defer s.Close()

	for {
		message, err := s.Read()

		if err != nil {
			return err
		}

		// Handle code response
		if message.MsgType == 1 {
			_ = s.Write(2, []byte("OK"))

			v := parseJagexState(string(message.Data))

			err := flow.Exchange(v.Get("code"), v.Get("state"))

			if err != nil {
				return err
			}

			break
		}
	}

	return startConsentHandler(flow)
}

func startConsentHandler(flow *auth.Flow) error {
	mux := http.NewServeMux()

	s := &http.Server{
		Addr:    ":80",
		Handler: mux,
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Handle data by sending it to /data with the #bla
		if r.URL.RawQuery == "" {
			w.Write([]byte(`
<script type="text/javascript">
if (window.location.hash) {
    const hashContent = window.location.hash.substring(1);
    
    const newUrl = window.location.origin + window.location.pathname + '?' + hashContent;
    
    window.location.replace(newUrl);
}
</script>
`))

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authentication successful! You may now close this window."))

		q := r.URL.Query()

		flow.Consent(q.Get("id_token"))

		time.AfterFunc(1*time.Second, func() {
			s.Shutdown(context.Background())
		})
	})

	return s.ListenAndServe()
}
