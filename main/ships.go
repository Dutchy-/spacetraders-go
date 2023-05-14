package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Dutchy-/spacetrader-go/client"
	intersect "github.com/juliangruber/go-intersect"
)

type Ship struct {
	client.Ship
	Cooldown client.Cooldown
}

type Miner struct {
	Ship
	State    MinerState
	Contract client.Contract
	// OreType  client.TradeSymbol
	Target client.Survey
}

type BaseShip interface {
	GoTo(waypoint *client.Waypoint)
	GoToSymbol(dest string)
	Status() client.ShipNavStatus
	Survey() []client.Survey
	Undock()
	Dock()
	GetCooldown() client.Cooldown
	ScanWaypoints() []client.ScannedWaypoint
	Refresh()
	SetCooldown(cooldown client.Cooldown)
	UpdateMarket() (client.Market, error)
	HasLowFuel() bool
	Refuel() client.Agent
}

type MinerShip interface {
	BaseShip
	Extract()
	Scan()
	Deliver() client.Contract
	Sell() (client.Agent, client.MarketTransaction)
	Run()
}

type MinerState string

const (
	DOCKED         MinerState = "DOCKED"
	ORBIT_STATION  MinerState = "ORBIT_STATION"
	ORBIT_ASTEROID MinerState = "ORBIT_ASTEROID"
	START_TRAVEL   MinerState = "START_TRAVEL"
	IN_TRANSIT     MinerState = "IN_TRANSIT"
	SURVEY         MinerState = "SURVEY"
	EXTRACT        MinerState = "EXTRACT"
	FIND_SELL      MinerState = "FIND_SELL"
	SELL_REMAINING MinerState = "SELL_REMAINING"
	UPDATE_MARKET  MinerState = "UPDATE_MARKET"
	JETTISON       MinerState = "JETTISON"
	REFUEL         MinerState = "REFUEL"
)

func NewCooldown(expiration time.Time) client.Cooldown {
	return client.Cooldown{
		Expiration:       expiration,
		RemainingSeconds: int(time.Until(expiration).Seconds()),
	}
}

func (ship *Ship) HasLowFuel() bool {
	return float64(ship.Fuel.Current) < (float64(ship.Fuel.Capacity) * 0.25)
}

func (ship *Ship) Refuel() client.Agent {
	resp, err := Client.RefuelShipWithResponse(context.TODO(), ship.Symbol)
	if err != nil {
		panic(err)
	}
	data := resp.JSON200.Data
	ship.Fuel = data.Fuel
	return data.Agent
}

func (ship *Ship) UpdateMarket() (client.Market, error) {
	resp, err := Client.GetMarketWithResponse(context.TODO(), ship.Nav.SystemSymbol, ship.Nav.WaypointSymbol)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode() != 200 {
		if resp.StatusCode() == 404 {
			return client.Market{}, errors.New("location does not have market")
		}
		panic(string(resp.Body))
	}
	return resp.JSON200.Data, nil
}

func (ship *Ship) Refresh() {
	resp, err := Client.GetShipNavWithResponse(context.TODO(), ship.Symbol)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode() != 200 {
		panic(string(resp.Body))
	}
	ship.Nav = resp.JSON200.Data
}

func (ship *Ship) IsFull() bool {
	return ship.Cargo.Capacity == ship.Cargo.Units
}

func (ship *Ship) ScanWaypoints() []client.ScannedWaypoint {
	resp, err := Client.CreateShipWaypointScanWithResponse(context.TODO(), ship.Symbol)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode() != 201 {
		panic(string(resp.Body))
	}
	data := resp.JSON201.Data
	ship.SetCooldown(data.Cooldown)
	return data.Waypoints
}

func (ship *Ship) GetCooldown() client.Cooldown {
	return ship.Cooldown
}

func (ship *Ship) SetCooldown(cooldown client.Cooldown) {
	log.Printf("Ship %s is in cooldown for %d seconds", ship.Symbol, cooldown.RemainingSeconds)
	ship.Cooldown = cooldown
}

func (ship *Ship) Status() client.ShipNavStatus {
	return ship.Nav.Status
}

func (ship *Ship) Undock() {
	resp, err := Client.OrbitShipWithResponse(context.TODO(), ship.Symbol)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode() != 200 {
		panic(string(resp.Body))
	}
	ship.Nav = resp.JSON200.Data.Nav
}

func (ship *Ship) Dock() {
	resp, err := Client.DockShipWithResponse(context.TODO(), ship.Symbol)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode() != 200 {
		panic(string(resp.Body))
	}
	ship.Nav = resp.JSON200.Data.Nav
}

func (ship *Ship) GoTo(waypoint *client.Waypoint) {
	ship.GoToSymbol(waypoint.Symbol)
}
func (ship *Ship) GoToSymbol(dest string) {
	resp, err := Client.NavigateShipWithResponse(context.TODO(), ship.Symbol, client.NavigateShipJSONRequestBody{
		WaypointSymbol: dest,
	})
	if err != nil {
		panic(err)
	}
	if resp.StatusCode() != 200 {
		panic(string(resp.Body))
	}
	data := resp.JSON200.Data
	ship.Nav = data.Nav
	ship.Fuel = data.Fuel
	ship.SetCooldown(NewCooldown(ship.Nav.Route.Arrival))
}

func (ship *Ship) Survey() []client.Survey {
	resp, err := Client.CreateSurveyWithResponse(context.TODO(), ship.Symbol)
	if err != nil || resp.StatusCode() != 201 {

		fmt.Println(err)
		fmt.Printf("%s\n", resp.Body)
	}
	data := resp.JSON201.Data
	ship.SetCooldown(data.Cooldown)
	return data.Surveys
}

func (ship *Miner) Sell(good client.TradeSymbol) (client.Agent, client.MarketTransaction) {
	for _, c := range ship.Cargo.Inventory {
		if c.Symbol == string(good) {
			resp, err := Client.SellCargoWithResponse(context.TODO(), ship.Symbol, client.SellCargoJSONRequestBody{
				Symbol: c.Symbol,
				Units:  c.Units,
			})
			if err != nil {
				panic(err)
			}
			if resp.StatusCode() != 201 {
				panic(string(resp.Body))
			}
			data := resp.JSON201.Data
			ship.Cargo = data.Cargo
			return data.Agent, data.Transaction
		}
	}
	// We only call this with a goods check so this does not happen
	return client.Agent{}, client.MarketTransaction{}
}

func (ship *Miner) Jettison(good client.TradeSymbol) {
	for _, c := range ship.Cargo.Inventory {
		if c.Symbol == string(good) {
			resp, err := Client.JettisonWithResponse(context.TODO(), ship.Symbol, client.JettisonJSONRequestBody{
				Symbol: c.Symbol,
				Units:  c.Units,
			})
			if err != nil {
				panic(err)
			}
			if resp.StatusCode() != 200 {
				panic(string(resp.Body))
			}
			data := resp.JSON200.Data
			ship.Cargo = data.Cargo
		}
	}
}

func (ship *Miner) Deliver() (client.Contract, error) {
	symbol := (*ship.Contract.Terms.Deliver)[0].TradeSymbol
	units := 0
	for _, c := range ship.Cargo.Inventory {
		if c.Symbol == symbol {
			units = c.Units
		}
	}
	if units != 0 {
		resp, err := Client.DeliverContractWithResponse(context.TODO(), ship.Contract.Id, client.DeliverContractJSONRequestBody{
			ShipSymbol:  ship.Symbol,
			TradeSymbol: symbol,
			Units:       units,
		})
		if err != nil {
			panic(err)
		}
		if resp.StatusCode() != 200 {
			panic(string(resp.Body))
		}
		data := resp.JSON200.Data
		ship.Contract = data.Contract
		ship.Cargo = data.Cargo
		return data.Contract, nil
	}
	return client.Contract{}, errors.New("nothing to deliver")
}

func (ship *Miner) Extract() *client.Extraction {
	resp, err := Client.ExtractResourcesWithResponse(context.TODO(), ship.Symbol, client.ExtractResourcesJSONRequestBody{
		Survey: &ship.Target,
	})
	if err != nil {
		log.Printf("Failed to extract: %v", err)
		return nil
	}
	data := resp.JSON201.Data
	ship.SetCooldown(data.Cooldown)
	ship.Cargo = data.Cargo
	return &data.Extraction
}

func (ship *Miner) HasContractGood() bool {
	for _, deliver := range *ship.Contract.Terms.Deliver {
		for _, ci := range ship.Cargo.Inventory {
			if deliver.TradeSymbol == ci.Symbol {
				return true
			}
		}
	}
	return false
}

func (ship *Miner) CanSellHere(market client.Market) bool {
	canSell := market.GetImportAndExchangeGoods()
	cargo := ship.Cargo.GetCargoGoodsExceptAntimatter()
	toSell := intersect.Hash(canSell, cargo)
	return len(toSell) > 0
}

func (ship *Miner) InitState() {
	switch ship.Status() {
	case client.DOCKED:
		ship.State = DOCKED
	case client.INORBIT:
		switch ship.Nav.Route.Destination.Type {
		case client.WaypointTypeASTEROIDFIELD:
			ship.State = ORBIT_ASTEROID
		case client.WaypointTypeORBITALSTATION:
			ship.State = ORBIT_STATION
		case client.WaypointTypePLANET:
			ship.State = ORBIT_STATION
		default:
			ship.State = ORBIT_STATION
		}
	case client.INTRANSIT:
		ship.State = IN_TRANSIT
	}

}

func (ship *Miner) Run(gameState *State) {
	if ship.State == "" {
		ship.InitState()
	}
	log.Printf("Miner %s in state %s\n", ship.Symbol, ship.State)
	switch ship.State {
	case REFUEL:
		beforeFuel := ship.Fuel.Current
		beforeCredits := gameState.Agent.Credits
		agent := ship.Refuel()
		gameState.Agent = agent
		afterFuel := ship.Fuel.Current
		afterCredits := gameState.Agent.Credits
		log.Printf("Bought %d fuel for %d credits, %d credits remaining", afterFuel-beforeFuel, beforeCredits-afterCredits, afterCredits)
		ship.State = DOCKED
	case DOCKED:
		if ship.IsFull() {
			contract, err := ship.Deliver()
			if err == nil {
				good := (*contract.Terms.Deliver)[0]
				log.Printf("Delivered %s, %d/%d fulfilled", good.TradeSymbol, good.UnitsFulfilled, good.UnitsRequired)
				gameState.Contracts[0] = contract
			}
		}
		ship.State = SELL_REMAINING
	case UPDATE_MARKET:
		market, err := ship.UpdateMarket()
		if err == nil {
			gameState.UpdateMarket(market)
		}
		ship.State = ORBIT_STATION
	case SELL_REMAINING:
		market := gameState.Markets[ship.Nav.WaypointSymbol]
		canSell := market.GetImportAndExchangeGoods()
		cargo := ship.Cargo.GetCargoGoodsExceptAntimatter()
		toSell := intersect.Hash(canSell, cargo)
		if len(toSell) > 0 {
			good, _ := toSell[0].(client.TradeSymbol)
			agent, trans := ship.Sell(good)
			gameState.Agent = agent
			log.Printf("Sold %d %s for %d credits", trans.Units, trans.TradeSymbol, trans.TotalPrice)
			log.Printf("Account now holds %d credits", agent.Credits)
		} else {
			ship.Undock()
			ship.State = ORBIT_STATION
		}
		// }
	case ORBIT_STATION:
		market, ok := gameState.Markets[ship.Nav.WaypointSymbol]
		if !ok && gameState.Waypoints[ship.Nav.WaypointSymbol].HasMarket() {
			ship.State = UPDATE_MARKET
		} else if (ship.HasContractGood() && ship.Nav.WaypointSymbol == (*ship.Contract.Terms.Deliver)[0].DestinationSymbol) || ship.CanSellHere(market) {
			ship.Dock()
			ship.State = DOCKED
		} else if ship.HasLowFuel() {
			ship.Dock()
			ship.State = REFUEL
		} else if len(ship.Cargo.Inventory) > 1 {
			ship.State = FIND_SELL
		} else {
			if gameState.GetAsteroid(ship.Nav.SystemSymbol) == nil {
				waypoints := ship.ScanWaypoints()
				gameState.AddWaypoints(ship.Nav.SystemSymbol, waypoints)
			}
			ship.State = START_TRAVEL
		}
	case START_TRAVEL:
		log.Printf("Ship %s leaving station with %d free cargo space", ship.Symbol, ship.Cargo.Capacity-ship.Cargo.Units)
		wp := gameState.GetAsteroid(ship.Nav.SystemSymbol)
		if ship.Nav.WaypointSymbol != wp.Symbol {
			ship.GoTo((*client.Waypoint)(wp))
		}
		ship.State = IN_TRANSIT
	case FIND_SELL:
		// sellCargo := ship.Cargo.GetCargoGoodsExceptAntimatter()
		waypoints, haveWaypoints := gameState.WaypointsBySystem[ship.Nav.SystemSymbol]
		if !haveWaypoints {
			waypoints := ship.ScanWaypoints()
			gameState.AddWaypoints(ship.Nav.SystemSymbol, waypoints)
		} else {
			found := false
			for _, wp := range waypoints {
				if wp.HasMarket() {
					market, haveMarket := gameState.Markets[wp.Symbol]
					if ship.Nav.WaypointSymbol != wp.Symbol && (!haveMarket || (haveMarket && ship.CanSellHere(market))) {
						found = true
						ship.GoTo((*client.Waypoint)(&wp))
						ship.State = IN_TRANSIT
						break
					} else if haveMarket && ship.CanSellHere(market) {
						found = true
						ship.Dock()
						ship.State = DOCKED
						break
					}
				}

			}
			if !found {
				ship.State = JETTISON
			}
		}
	case JETTISON:
		log.Println("We need to jettison the remaining cargo")
		cargo := ship.Cargo.GetCargoGoodsExceptAntimatter()
		if len(cargo) > 0 {
			ship.Jettison(cargo[0])
			log.Printf("Jettisoned all %s", cargo[0])
		} else {
			ship.State = ORBIT_STATION
		}
	case ORBIT_ASTEROID:
		if ship.IsFull() && ship.HasContractGood() {
			dest := (*ship.Contract.Terms.Deliver)[0].DestinationSymbol
			ship.GoToSymbol(dest)
			ship.State = IN_TRANSIT
		} else if ship.IsFull() {
			ship.State = FIND_SELL
		} else {
			survey := gameState.GetOreSurvey(ship.Nav.WaypointSymbol, (*ship.Contract.Terms.Deliver)[0].TradeSymbol)
			if survey == nil {
				ship.State = SURVEY
			} else {
				log.Printf("Targetting survey %s, which is valid for %d", survey.Symbol, int(time.Until(survey.Expiration).Seconds()))
				ship.Target = *survey
				ship.State = EXTRACT
			}
		}
	case SURVEY:
		surveys := ship.Survey()
		gameState.AddSurveys(ship.Nav.WaypointSymbol, surveys)
		for _, survey := range surveys {
			for _, dep := range survey.Deposits {
				fmt.Println(dep.Symbol, survey.Symbol)
			}
		}
		ship.State = ORBIT_ASTEROID
	case EXTRACT:
		if ship.Target.Expiration.After(time.Now().Add(time.Second)) {
			e := ship.Extract()
			log.Printf("Miner %s extracted %d %s\n", ship.Symbol, e.Yield.Units, e.Yield.Symbol)
		} else {
			ship.State = ORBIT_ASTEROID
		}
		if ship.Cargo.Capacity == ship.Cargo.Units {
			ship.State = ORBIT_ASTEROID
		}
	case IN_TRANSIT:
		// do nothing?
		ship.Refresh()
		ship.InitState()
	}
}
