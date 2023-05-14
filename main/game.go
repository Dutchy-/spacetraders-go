package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/Dutchy-/spacetrader-go/client"
)

const STATE_FILE = "game.state.json"

type Game struct {
	// Client *client.ClientWithResponses
	State State `json:"state"`
}

type State struct {
	Agent             client.Agent                        `json:"agent"`
	Contracts         []client.Contract                   `json:"contracts"`
	Ships             []BaseShip                          `json:"-"`
	Surveys           map[string][]client.Survey          `json:"surveys"`
	WaypointsBySystem map[string][]client.ScannedWaypoint `json:"waypoints_by_system"`
	Waypoints         map[string]client.ScannedWaypoint   `json:"waypoints"`
	Markets           map[string]client.Market            `json:"markets"`
}

func NewGame() *Game {
	game := Game{}
	b, err := os.ReadFile(STATE_FILE)
	if err == nil {
		err = json.Unmarshal(b, &game)
		if err != nil {
			panic(err)
		}
		if game.State.WaypointsBySystem == nil {
			game.State.WaypointsBySystem = make(map[string][]client.ScannedWaypoint)
		}
	} else {
		game.State.Surveys = make(map[string][]client.Survey)
		game.State.WaypointsBySystem = make(map[string][]client.ScannedWaypoint)
		game.State.Waypoints = make(map[string]client.ScannedWaypoint)
		game.State.Markets = make(map[string]client.Market)
	}
	return &game
}

func (game *Game) Run() {

	game.InitAgent()
	game.InitContracts()

	// fmt.Println("Agent: ", agent.JSON200.Data.Symbol, agent.JSON200.Data.AccountId, agent.JSON200.Data.Credits, agent.JSON200.Data.Headquarters)

	game.InitShips()

	if !game.State.Contracts[0].Accepted {
		resp, err := Client.AcceptContractWithResponse(context.TODO(), game.State.Contracts[0].Id)
		if err != nil {
			panic(err)
		}
		if resp.StatusCode() != 200 {
			panic(string(resp.Body))
		}
		data := resp.JSON200.Data
		game.State.Agent = data.Agent
		game.State.Contracts[0] = data.Contract
	}

	// assign contract to ship, single ship for now
	m, _ := game.State.Ships[0].(*Miner)
	m.Contract = game.State.Contracts[0]

	// background-save every 10 seconds
	go func() {
		for {
			b, err := json.MarshalIndent(*game, "", "  ")
			if err != nil {
				panic(err)
			}
			err = os.WriteFile(STATE_FILE, b, 0644)
			if err != nil {
				panic(err)
			}
			time.Sleep(10 * time.Second)
		}
	}()
	// main game loop
	for {
		allOnCooldown := true
		for _, ship := range game.State.Ships {
			cooldown := ship.GetCooldown()
			if (cooldown == client.Cooldown{} || cooldown.Expiration.Before(time.Now())) {
				allOnCooldown = false
				switch ship := ship.(type) {
				case *Miner:
					ship.Run(&game.State)
				}
			}
		}
		if allOnCooldown {
			// fmt.Println("All ships on cooldown, waiting...")
			time.Sleep(500 * time.Millisecond)
		}
	}

	// limit := 20
	// end := false
	// for page := 1; !end; page++ {
	// 	systems, _ := Client.GetSystemsWithResponse(context.TODO(), &client.GetSystemsParams{Page: &page, Limit: &limit})
	// 	total := systems.JSON200.Meta.Total
	// 	for _, system := range systems.JSON200.Data {
	// 		fmt.Println(system.Symbol, system.SectorSymbol, system.Type, system.X, system.Y)
	// 	}
	// 	end = limit*(page) >= total
	// }

	// system, _ := Client.GetSystemWithResponse(context.TODO(), ships.JSON200.Data[0].Nav.SystemSymbol)
	// fmt.Println("Current System: ", system.JSON200.Data.Symbol, system.JSON200.Data.SectorSymbol, system.JSON200.Data.Factions)
}

func (game *Game) InitShips() {
	log.Println("Initialising Ships...")
	game.State.Ships = make([]BaseShip, 0)
	ships, _ := Client.GetMyShipsWithResponse(context.TODO(), &client.GetMyShipsParams{})
	for _, ship := range ships.JSON200.Data {
		game.State.Ships = append(game.State.Ships, &Miner{Ship: Ship{Ship: ship}})
	}
}

func (game *Game) InitContracts() {
	log.Println("Initialising Contracts...")
	contracts, _ := Client.GetContractsWithResponse(context.TODO(), &client.GetContractsParams{})
	game.State.Contracts = contracts.JSON200.Data
	Pprint(game.State.Contracts)
}

func (game *Game) InitAgent() {
	log.Println("Initialising Agent...")
	agent, err := Client.GetMyAgentWithResponse(context.TODO())
	if err != nil {
		panic(err)
	}
	if agent.StatusCode() != 200 {
		panic(string(agent.Body))
	}
	game.State.Agent = agent.JSON200.Data
}

func (state *State) AddSurveys(waypointSymbol string, surveys []client.Survey) {
	state.Surveys[waypointSymbol] = append(state.Surveys[waypointSymbol], surveys...)
}

func (state *State) AddWaypoints(systemSymbol string, waypoints []client.ScannedWaypoint) {
	state.WaypointsBySystem[systemSymbol] = append(state.WaypointsBySystem[systemSymbol], waypoints...)
	for _, wp := range waypoints {
		state.Waypoints[wp.Symbol] = wp
	}
}

func (state *State) GetAsteroid(systemSymbol string) *client.ScannedWaypoint {
	for _, wp := range state.WaypointsBySystem[systemSymbol] {
		if wp.Type == client.WaypointTypeASTEROIDFIELD {
			return &wp
		}
	}
	return nil
}

func (state *State) GetOreSurvey(waypointSymbol string, oreType string) *client.Survey {
	for _, survey := range state.Surveys[waypointSymbol] {
		if survey.Expiration.After(time.Now()) {
			for _, dep := range survey.Deposits {
				if dep.Symbol == oreType {
					return &survey
				}
			}
		}
	}
	return nil
}

func (state *State) UpdateMarket(market client.Market) {
	state.Markets[market.Symbol] = market
}
