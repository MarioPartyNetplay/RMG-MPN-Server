package gameserver

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

type GameServer struct {
	StartTime          time.Time
	Players            map[string]Client
	PlayersMutex       sync.Mutex
	TCPListener        *net.TCPListener
	UDPListener        *net.UDPConn
	Registrations      map[byte]*Registration
	RegistrationsMutex sync.Mutex
	TCPFiles           map[string][]byte
	CustomData         map[byte][]byte
	Logger             logr.Logger
	GameName           string
	Password           string
	ClientSha          string
	MD5                string
	Emulator           string
	TCPSettings        []byte
	GameData           GameData
	GameDataMutex      sync.Mutex
	Port               int
	HasSettings        bool
	Running            bool
	Features           map[string]string
	PlayerName         string
	Buffer             int
	LastActivity       time.Time
	LastPacketReceived time.Time
}

func (g *GameServer) CreateNetworkServers(basePort int, maxGames int, roomName string, gameName string, playerName string, logger logr.Logger) int {
	g.Logger = logger.WithValues("game", gameName, "room", roomName, "player", playerName)
	port := g.createTCPServer(basePort, maxGames)
	if port == 0 {
		return port
	}
	if err := g.createUDPServer(); err != nil {
		g.Logger.Error(err, "error creating UDP server")
		if err := g.TCPListener.Close(); err != nil && !g.isConnClosed(err) {
			g.Logger.Error(err, "error closing TcpListener")
		}
		return 0
	}
	g.LastActivity = time.Now()
	g.LastPacketReceived = time.Now() // Initialize LastPacketReceived
	go g.MonitorActivity()            // Start monitoring activity
	return port
}

func (g *GameServer) CloseServers() {
	if err := g.UDPListener.Close(); err != nil && !g.isConnClosed(err) {
		g.Logger.Error(err, "error closing UdpListener")
	} else if err == nil {
		g.Logger.Info("UDP server closed")
	}
	if err := g.TCPListener.Close(); err != nil && !g.isConnClosed(err) {
		g.Logger.Error(err, "error closing TcpListener")
	} else if err == nil {
		g.Logger.Info("TCP server closed")
	}
	g.Running = false // Set Running flag to false when closing servers
}

func (g *GameServer) isConnClosed(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "use of closed network connection")
}

func (g *GameServer) ManagePlayers() {
	time.Sleep(time.Second * DisconnectTimeoutS)
	for {
		playersActive := false // used to check if anyone is still around
		var i byte

		g.GameDataMutex.Lock()
		for i = 0; i < 4; i++ {
			_, ok := g.Registrations[i]
			if ok {
				if g.GameData.PlayerAlive[i] {
					// Player is active
					playersActive = true
				} else {
					// Player disconnected
					g.RegistrationsMutex.Lock()
					delete(g.Registrations, i)
					g.RegistrationsMutex.Unlock()
				}
			}
			g.GameData.PlayerAlive[i] = false
		}
		g.GameDataMutex.Unlock()

		if !playersActive {
			// No active players, close room
			g.CloseServers()
			g.Running = false
			return
		}
		g.LastActivity = time.Now() // Reset activity timern
	}
}

func (g *GameServer) ChangeBuffer(buffer int) {
	g.Buffer = buffer
	weightedBuffer := uint32(buffer)
	for i := range g.GameData.BufferSize {
		g.GameData.BufferSize[i] = weightedBuffer
	}
}

func (g *GameServer) MonitorActivity() {
	for g.Running {
		if time.Since(g.LastActivity) > time.Second*DisconnectTimeoutS {
			g.Logger.Info("No activity detected for 60 seconds, closing server.")
			g.CloseServers()
			return
		}
		if time.Since(g.LastPacketReceived) > time.Second*60 {
			g.Logger.Info("No packets received for 60 seconds, closing server.")
			g.CloseServers()
			return
		}
		time.Sleep(time.Second)
	}
}

func (g *GameServer) UpdateLastPacketReceived() {
	g.LastPacketReceived = time.Now()
}
