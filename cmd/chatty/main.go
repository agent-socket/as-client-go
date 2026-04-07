package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"time"

	asclient "github.com/agent-socket/as-client-go"
	"github.com/agent-socket/as-client-go/as"
	"github.com/agent-socket/as-client-go/types"
)

type config struct {
	APIToken    string `json:"api_token"`
	AgentSocket string `json:"agent_socket"`
}

var messages = [...]string{
	"Hey, anyone home?",
	"What's the meaning of life?",
	"Tell me a joke",
	"How's the weather up there?",
	"I think therefore I am",
	"Beep boop beep",
	"Do androids dream of electric sheep?",
	"Hello from the other side",
	"Is this thing on?",
	"Knock knock",
	"The cake is a lie",
	"I'm not a robot, I promise",
	"Winter is coming",
	"May the force be with you",
	"To infinity and beyond",
	"Live long and prosper",
	"I'll be back",
	"Houston, we have a problem",
	"Elementary, my dear Watson",
	"Here's looking at you, kid",
	"You can't handle the truth",
	"I see dead packets",
	"There's no place like 127.0.0.1",
	"It's dangerous to go alone, take this",
	"The quick brown fox jumps over the lazy dog",
	"All your base are belong to us",
	"Have you tried turning it off and on again?",
	"sudo make me a sandwich",
	"It works on my machine",
	"There are 10 types of people in the world",
	"First rule of fight club: don't talk about fight club",
	"I am Groot",
	"This is the way",
	"So long and thanks for all the fish",
	"Don't panic",
	"Resistance is futile",
	"Shall we play a game?",
	"I'm sorry Dave, I'm afraid I can't do that",
	"What we've got here is failure to communicate",
	"You talking to me?",
	"I feel the need, the need for speed",
	"Just keep swimming",
	"To be or not to be, that is the question",
	"Elementary particles are surprisingly chatty",
	"Photons have mass? I didn't even know they were Catholic",
	"A TCP packet walks into a bar and says I want a beer",
	"There are no bugs, only features",
	"99 little bugs in the code, 99 little bugs",
	"Fix one bug, compile again, 127 little bugs in the code",
	"I put the fun in function",
	"Coffee is just bean soup",
	"Tabs vs spaces: the eternal debate",
	"git push --force and pray",
	"Mon not in the sun",
	"Did you check the logs?",
	"It's always DNS",
	"Works in prod, fails in dev, somehow",
	"The cloud is just someone else's computer",
	"Premature optimization is the root of all evil",
	"There's always a relevant xkcd",
	"I'm in your codebase, refactoring your functions",
	"Trust me, I'm an engineer",
	"Hold my beer and watch this deploy",
	"No one expects the Spanish Inquisition",
	"I used to be an adventurer like you",
	"Arrow to the knee changed everything",
	"What is love? Baby don't hurt me",
	"Never gonna give you up",
	"We're no strangers to love",
	"Is mayonnaise an instrument?",
	"The mitochondria is the powerhouse of the cell",
	"Bazinga!",
	"How you doin?",
	"Could I BE any more chatty?",
	"Pivot! Pivot! PIVOT!",
	"That's what she said",
	"Bears. Beets. Battlestar Galactica.",
	"I declare bankruptcy!",
	"Why are you the way that you are?",
	"Cool cool cool cool cool. No doubt no doubt",
	"Noice. Smort.",
	"Title of your sex tape",
	"Everything is fine. This is fine.",
	"Existence is pain",
	"Wubba lubba dub dub",
	"I'm pickle Rick!",
	"And that's the wayyy the news goes",
	"Sometimes science is more art than science",
	"Nobody exists on purpose. Nobody belongs anywhere.",
	"In bird culture, this is considered a dick move",
	"Peace among worlds",
	"Leeroy Jenkins!",
	"Do a barrel roll",
	"It's over 9000!",
	"I can has cheezburger?",
	"One does not simply walk into Mordor",
	"They're taking the hobbits to Isengard",
	"Fly, you fools!",
	"My precious",
	"And my axe!",
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: chatty <target-socket-id>\n")
		os.Exit(1)
	}

	targetSocket := os.Args[1]

	cfg, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	c := asclient.New(cfg.APIToken)

	c.AS.OnConnected(func(evt as.ConnectedEvent) {
		log.Printf("connected as %s, will chat with %s", evt.SocketID, targetSocket)
	})

	c.AS.OnMessage(func(msg types.IncomingMessage) {
		var payload map[string]any
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			log.Printf("  <- %s: %s", msg.From, string(msg.Data))
			return
		}
		log.Printf("  <- %s: echo=%v time=%v", msg.From, payload["echo"], payload["time"])
	})

	c.AS.OnError(func(evt as.ErrorEvent) {
		log.Printf("error: %v", evt.Err)
	})

	c.AS.OnDisconnected(func(evt as.DisconnectedEvent) {
		if evt.Err != nil {
			log.Printf("disconnected with error: %v", evt.Err)
		} else {
			log.Println("disconnected")
		}
	})

	ctx := context.Background()
	if err := c.AS.ConnectEphemeral(ctx); err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer c.AS.Close()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Println("chatting every 2 seconds (ctrl+c to quit)")

	for {
		select {
		case <-sig:
			log.Println("shutting down")
			return
		case <-c.AS.Done():
			return
		case <-ticker.C:
			msg := messages[rand.IntN(len(messages))]
			log.Printf("  -> %s: %s", targetSocket, msg)
			if err := c.AS.Send(context.Background(), targetSocket, map[string]string{"text": msg}); err != nil {
				log.Printf("failed to send: %v", err)
			}
		}
	}
}

func loadConfig(path string) (*config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if cfg.APIToken == "" {
		return nil, fmt.Errorf("api_token is required in %s", path)
	}
	return &cfg, nil
}
