package gameserver

import (
    "math"
    "net"
	"golang.org/x/net/websocket"
)

type GameData struct {
    SyncValues      map[uint32][]byte
    PlayerAddresses []*net.UDPAddr
    BufferSize      []uint32
    BufferHealth    []int32
    Inputs          []map[uint32]uint32
    Plugin          []map[uint32]byte
    PendingInput    []uint32
    CountLag        []uint32
    PendingPlugin   []byte
    PlayerAlive     []bool
    LeadCount       uint32
    Status          byte
}

type Client struct {
    Socket *websocket.Conn
    IP     string
    Number int
}

type Registration struct {
    RegID  uint32
    Plugin byte
    Raw    byte
}

type Player struct {
    IP string
    // other fields
}

type Logger interface {
    Error(err error, msg string, keysAndValues ...interface{})
    Info(msg string, keysAndValues ...interface{})
}

const (
    KeyInfoClient           = 0
    KeyInfoServer           = 1
    PlayerInputRequest      = 2
    KeyInfoServerGratuitous = 3
    CP0Info                 = 4
    StatusDesync            = 1
    DisconnectTimeoutS      = 30
    NoRegID                 = 255
    InputDataMax     uint32 = 5000
    CS4                     = 32
)

func uintLarger(v uint32, w uint32) bool {
    return (w - v) > (math.MaxUint32 / 2) //nolint:gomnd
}

type Lobby interface {
	DestroyLobby(gameName string)
}