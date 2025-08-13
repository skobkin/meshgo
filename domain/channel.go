package domain

// Channel represents a Meshtastic channel and its encryption class.
type Channel struct {
	Name     string
	PSKClass int // 0=none,1=default,2=custom
}
