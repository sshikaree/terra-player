package main

import (
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	vlc_addr = "localhost:9999"
)

var headers = map[string][]string{
	"User-Agent":      {"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/30.0.1599.114 Chrome/30.0.1599.114 Safari/537.36"},
	"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"},
	"Accept-Language": {"ru-RU,ru;q=0.8,en-US;q=0.6,en;q=0.4"},
}

// Player commands
var commands = map[string]string{
	"play":   "play <genre> --- play selected genre (or random, if not defined)",
	"stop":   "stop ----------- stop stream",
	"resume": "resume --------- continue playing stopped stream",
	"exit":   "exit ----------- close program",
	// "next":      "next",
	// "prev":      "prev",
	"help":      "help ----------- show this help",
	"get_title": "get_title ------ the title of the current stream",
	// "clear":     "clear",
	"genres":  "genres --------- get list of available genres",
	"volup":   "volup <val> ---- raise audio volume <val> steps",
	"voldown": "voldown <val> -- lower audio volume <val> steps",
	"info":    "info ----------- information about the current stream",
}

// Colors for output
var colors = map[string]string{
	// Regular
	"Black":  "\033[0;30m",
	"Red":    "\033[0;31m",
	"Green":  "\033[0;32m",
	"Yellow": "\033[0;33m",
	"Blue":   "\033[0;34m",
	"Purple": "\033[0;35m",
	"Cyan":   "\033[0;36m",
	"White":  "\033[0;37m",
	// Bold
	"BBlack":  "\033[1;30m",
	"BRed":    "\033[1;31m",
	"BGreen":  "\033[1;32m",
	"BYellow": "\033[1;33m",
	"BBlue":   "\033[1;34m",
	"BPurple": "\033[1;35m",
	"BCyan":   "\033[1;36m",
	"BWhite":  "\033[1;37m",
	// High Intensity
	"IBlack":  "\033[0;90m",
	"IRed":    "\033[0;91m",
	"IGreen":  "\033[0;92m",
	"IYellow": "\033[0;93m",
	"IBlue":   "\033[0;94m",
	"IPurple": "\033[0;95m",
	"ICyan":   "\033[0;96m",
	"IWhite":  "\033[0;97m",
	// Bold High Intensity
	"BIBlack":  "\033[1;90m",
	"BIRed":    "\033[1;91m",
	"BIGreen":  "\033[1;92m",
	"BIYellow": "\033[1;93m",
	"BIBlue":   "\033[1;94m",
	"BIPurple": "\033[1;95m",
	"BICyan":   "\033[1;96m",
	"BIWhite":  "\033[1;97m",

	// Reset
	"Reset": "\033[0m",
}

type Station struct {
	Name  string `xml:"name,attr"`
	ID    string `xml:"id,attr"`
	Genre string `xml:"genre,attr"`
}

type Player struct {
	conn     net.Conn
	vlc_addr string
}

func (p *Player) GetGenres() []string {
	addr := "http://yp.shoutcast.com/sbin/newxml.phtml"
	client := &http.Client{}
	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header = headers
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	page, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	type Genre struct {
		Name string `xml:"name,attr"`
	}

	type Genres struct {
		Genre []Genre `xml:"genre"`
	}
	genres := new(Genres)

	err = xml.Unmarshal(page, &genres)
	if err != nil {
		log.Fatal(err)
	}
	var list []string
	for _, g := range genres.Genre {
		list = append(list, g.Name)
	}
	return list

}

func (p *Player) GetStations(genre string) []Station {
	v := url.Values{}
	v.Set("search", genre)
	client := &http.Client{}
	addr := "http://yp.shoutcast.com/sbin/newxml.phtml?" + v.Encode()
	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header = headers
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	page, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	type Stations struct {
		Station []Station `xml:"station"`
	}

	stations := new(Stations)
	err = xml.Unmarshal(page, &stations)
	if err != nil {
		log.Fatal(err)
	}

	return stations.Station
}

func (p *Player) PlayRandom(stations []Station) {
	station := stations[rand.Intn(len(stations)-1)]
	id := station.ID
	if id == "" {
		fmt.Println("Error occured, stream was not found..")
		return
	}
	url := "http://yp.shoutcast.com/sbin/tunein-station.pls?id=" + id
	p.SendCommandToVLC(fmt.Sprintf("add %s\n", url))
	fmt.Printf("Connected to %s%s%s station\n", colors["Green"], station.Name, colors["Reset"])

}

func (p *Player) StartVLC() {
	_, err := exec.LookPath("cvlc")
	if err != nil {
		log.Fatal("Error: VLC was not found in your system.")
	}
	cmd := exec.Command(
		"cvlc",
		"--intf",
		"rc",
		"--rc-host",
		p.vlc_addr,
	)
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
}

func (p *Player) ConnectToVLC() {
	var err error
	for i := 0; i < 30; i++ {
		time.Sleep(200 * time.Millisecond)
		p.conn, err = net.Dial("tcp", p.vlc_addr)
		if err == nil {
			return
		}
	}
	log.Fatal("Error connecting to VLC")
}

func (p *Player) SendCommandToVLC(command string) {

	p.conn.Write([]byte(command))
}

func (p *Player) TCPListner() {
	r := bufio.NewReader(p.conn)
	// Drop VLC greeting
	r.ReadString('\n')
	r.ReadString('\n')
	//
	for {
		s, err := r.ReadString('\n')
		if err != nil {
			log.Println(err)
			break
		}
		fmt.Printf(s)
		fmt.Printf(colors["BGreen"] + "~~> " + colors["Reset"])

	}
}

func (p *Player) Close() {
	p.SendCommandToVLC("shutdown\n")
	p.conn.Close()
}

func NewPlayer(vlc_addr string) *Player {
	p := new(Player)
	p.vlc_addr = vlc_addr
	p.StartVLC()
	p.ConnectToVLC()
	go p.TCPListner()
	return p
}

func main() {
	log.SetFlags(log.Lshortfile)

	player := NewPlayer(vlc_addr)
	// Clean up on exit
	defer player.Close()
	// defer fmt.Println(colors["BRed"] + "\nPlease, don't forget to kill all VLC players! Sorry for this bug. " + colors["Reset"] + "Bye ;)")

	fmt.Println(colors["Yellow"] + "Command line radio player. Enter 'help' for commands, 'genres' for genres list." + colors["Reset"])

	rand.Seed(time.Now().UnixNano())

	flag.Parse()
	genre := flag.Arg(0)
	if genre != "" {
		stations := player.GetStations(genre)
		if len(stations) > 0 {
			player.PlayRandom(stations)
		}
	}

MainLoop:
	for {
		fmt.Printf(colors["BGreen"] + "~~> " + colors["Reset"])
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		cmd := strings.SplitN(strings.Trim(scanner.Text(), " "), " ", 2)

		// Debug
		//fmt.Println(cmd)

		if _, ok := commands[cmd[0]]; !ok {
			fmt.Println("Wrong command")
			continue
		}
		switch cmd[0] {
		case "play":
			if len(cmd) > 1 {
				genre = strings.Trim(cmd[1], "\" ")
			} else {
				genre = "random"
			}
			stations := player.GetStations(genre)
			if len(stations) > 0 {
				player.PlayRandom(stations)
			} else {
				fmt.Println("Can't get list of stations")
			}
		case "exit":
			player.SendCommandToVLC("clear\n")
			// SendCommandToVLC("shutdown\n")
			// os.Exit(0)
			break MainLoop

		case "stop":
			player.SendCommandToVLC("stop\n")
		case "resume":
			player.SendCommandToVLC("play\n")
		case "help":
			for _, helpline := range commands {
				fmt.Println(helpline)
			}
			continue
		case "clear":
			player.SendCommandToVLC("clear\n")
		case "genres":
			fmt.Println(colors["BBlack"] + "Plase, wait..." + colors["Reset"])
			genres := player.GetGenres()
			fmt.Println("List of available genres")
			fmt.Println(colors["Yellow"])
			for i, g := range genres {
				if i > 1 && (i+1)%3 == 0 {
					fmt.Printf("%-28s\n", g)
				} else {
					fmt.Printf("%-28s", g)
				}
			}
			fmt.Println(colors["Reset"])
		case "volup", "voldown":
			if len(cmd) > 1 && cmd[1] != "" {
				_, err := strconv.Atoi(cmd[1])
				if err != nil {
					fmt.Println("Wrong value")
					continue
				}
				player.SendCommandToVLC(fmt.Sprintf("%s %s\n", cmd[0], cmd[1]))
			} else {
				player.SendCommandToVLC(fmt.Sprintf("%s\n", cmd[0]))
			}
		case "get_title":
			player.SendCommandToVLC("get_title\n")
		case "info":
			player.SendCommandToVLC("info\n")

		}

	}

}
