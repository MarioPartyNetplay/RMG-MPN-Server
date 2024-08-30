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
	CreationTime       time.Time
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
	if g.UDPListener != nil {
		if err := g.UDPListener.Close(); err != nil && !g.isConnClosed(err) {
			g.Logger.Error(err, "error closing UdpListener")
		} else if err == nil {
			g.Logger.Info("UDP server closed")
		}
		g.UDPListener = nil // Ensure the UDPListener is set to nil after closing
	}
	if g.TCPListener != nil {
		if err := g.TCPListener.Close(); err != nil && !g.isConnClosed(err) {
			g.Logger.Error(err, "error closing TcpListener")
		} else if err == nil {
			g.Logger.Info("TCP server closed")
		}
		g.TCPListener = nil // Ensure the TCPListener is set to nil after closing
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

		g.GameDataMutex.Lock() // PlayerAlive and Status can be modified by processUDP in a different thread
		for i = 0; i < 4; i++ {
			_, ok := g.Registrations[i]
			if ok {
				if g.GameData.PlayerAlive[i] {
					g.Logger.Info("player status", "player", i, "regID", g.Registrations[i].RegID, "bufferSize", g.GameData.BufferSize[i], "bufferHealth", g.GameData.BufferHealth[i], "countLag", g.GameData.CountLag[i], "address", g.GameData.PlayerAddresses[i])
					playersActive = true
				} else {
					g.Logger.Info("play disconnected UDP", "player", i, "regID", g.Registrations[i].RegID, "address", g.GameData.PlayerAddresses[i])
					g.GameData.Status |= (0x1 << (i + 1)) //nolint:gomnd,mnd

					g.RegistrationsMutex.Lock() // Registrations can be modified by processTCP
					delete(g.Registrations, i)
					g.RegistrationsMutex.Unlock()
				}
			}
			g.GameData.PlayerAlive[i] = false
		}
		g.GameDataMutex.Unlock()

		if !playersActive {
			g.Logger.Info("no more players, closing room", "numPlayers", len(g.Players), "playTime", time.Since(g.StartTime).String(), "emulator", g.Emulator)
			g.CloseServers()
			g.Running = false
			return
		}
		time.Sleep(time.Second * DisconnectTimeoutS)
	}
}

func (g *GameServer) ChangeBuffer(buffer int) {
	g.Buffer = buffer
	weightedBuffer := uint32(buffer)
	for i := range g.GameData.BufferSize {
		g.GameData.BufferSize[i] = 0
		g.GameData.BufferHealth[i] = 0 + weightedBuffer
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
		g.PlayersMutex.Lock()
		noPlayers := len(g.Players) == 0
		g.PlayersMutex.Unlock()
		if noPlayers {
			g.Logger.Info("No players online, restarting server.")
			g.CloseServers()
			time.Sleep(time.Second * 10) // Wait for 10 seconds before restarting
			g.CreateNetworkServers(g.Port, 1, g.GameName, g.GameName, g.PlayerName, g.Logger)
		}
		time.Sleep(time.Minute * 2) // Check every 10 minutes
	}
}

func (g *GameServer) UpdateLastPacketReceived() {
	g.LastPacketReceived = time.Now()
}
