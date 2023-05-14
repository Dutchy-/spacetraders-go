package client

func (market Market) GetImportAndExchangeGoods() []TradeSymbol {
	goods := []TradeSymbol{}
	for _, good := range market.Imports {
		goods = append(goods, good.Symbol)
	}
	for _, good := range market.Exchange {
		goods = append(goods, good.Symbol)
	}
	return goods
}

func (cargo ShipCargo) GetCargoGoods() []TradeSymbol {
	goods := []TradeSymbol{}
	for _, c := range cargo.Inventory {
		goods = append(goods, TradeSymbol(c.Symbol))
	}
	return goods
}

func (cargo ShipCargo) GetCargoGoodsExceptAntimatter() []TradeSymbol {
	goods := []TradeSymbol{}
	for _, c := range cargo.Inventory {
		if c.Symbol != string(TradeSymbolANTIMATTER) {
			goods = append(goods, TradeSymbol(c.Symbol))
		}
	}
	return goods
}

func (wp ScannedWaypoint) HasMarket() bool {
	for _, trait := range wp.Traits {
		if trait.Symbol == WaypointTraitSymbolMARKETPLACE {
			return true
		}
	}
	return false
}
