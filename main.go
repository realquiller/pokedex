package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"pokedex/internal/pokecache"
	"strings"
	"time"
)

type cliCommand struct {
	name        string
	description string
	callback    func(args []string, config *Config) error
}

type Config struct {
	Next          string
	Previous      string
	Cache         *pokecache.Cache
	Areas         map[string]string //key = area name, value = area url
	CaughtPokemon map[string]Pokemon
}

type Pokemon struct {
	Name           string `json:"name"`
	BaseExperience int    `json:"base_experience"`
	Height         int    `json:"height"`
	Weight         int    `json:"weight"`
	Stats          []struct {
		BaseStat int `json:"base_stat"`
		Stat     struct {
			Name string `json:"name"` // e.g. "hp", "attack", etc.
		} `json:"stat"`
	} `json:"stats"`
	Types []struct {
		Type struct {
			Name string `json:"name"` // e.g. "normal", "flying", etc.
		} `json:"type"`
	} `json:"types"`
}

type LocationAreaList struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"results"`
}

var commands = make(map[string]cliCommand)

func cleanInput(text string) []string {
	// lowercase the text
	text = strings.ToLower(text)

	// use filter to remove whitespaces and put it into a slice
	return strings.Fields(text)
}

func loadAllAreas(config *Config) error {
	url := "https://pokeapi.co/api/v2/location-area/"
	config.Areas = make(map[string]string)

	for url != "" {
		if data, ok := config.Cache.Get(url); ok {
			var result LocationAreaList
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("failed to parse cached area data: %w", err)
			}
			for _, area := range result.Results {
				config.Areas[area.Name] = area.URL
			}
			url = result.Next
			continue
		}

		res, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to fetch area data: %w", err)
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		if res.StatusCode > 299 {
			return fmt.Errorf("bad status code: %d\nBody: %s", res.StatusCode, body)
		}

		var result LocationAreaList
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse area data: %w", err)
		}

		for _, area := range result.Results {
			config.Areas[area.Name] = area.URL
		}

		url = result.Next
	}

	return nil
}

func commandExit(args []string, config *Config) error {
	fmt.Print("Closing the Pokedex... Goodbye!\n")
	os.Exit(0)
	return nil
}

func commandHelp(args []string, config *Config) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage: ")
	for i := range commands {
		cmd := commands[commands[i].name]
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}
	return nil
}

func commandMap(args []string, config *Config) error {
	if config.Next == "" {
		config.Next = "https://pokeapi.co/api/v2/location-area/"
	}
	return showAreas(config.Next, config, false)
}

func commandMapB(args []string, config *Config) error {
	if config.Previous == "" {
		return fmt.Errorf("no previous page available")
	}
	return showAreas(config.Previous, config, true)
}

func commandCatch(args []string, config *Config) error {
	if len(args) == 0 {
		return fmt.Errorf("missing Pokémon name. Usage: catch <pokemon-name>")
	}

	pokemon_name := args[0]
	fmt.Printf("Throwing a Pokeball at %s...\n", pokemon_name)
	pokemon, err := getPokemonInfo(pokemon_name)
	if err != nil {
		return fmt.Errorf("failed to get pokemon info: %w", err)
	}

	catchRate := calculateCatchRate(pokemon.BaseExperience)

	caught := rand.Intn(100) < catchRate

	if caught {
		fmt.Printf("%s was caught!\n", pokemon.Name)
		fmt.Println("You may now inspect it with the inspect command.")
		config.CaughtPokemon[pokemon.Name] = *pokemon
	} else {
		fmt.Printf("%s escaped!\n", pokemon.Name)
	}
	return nil

}

func calculateCatchRate(baseExp int) int {
	rate := 100 - baseExp/3

	if rate < 5 {
		return 5
	}

	if rate > 90 {
		return 90
	}

	return rate
}

func getPokemonInfo(pokemonName string) (*Pokemon, error) {
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s", strings.ToLower(pokemonName))

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get pokemon info: %s", resp.Status)
	}

	var pokemon Pokemon

	err = json.NewDecoder(resp.Body).Decode(&pokemon)
	if err != nil {
		return nil, err
	}

	return &pokemon, nil
}

func commandInspect(args []string, config *Config) error {
	if len(args) == 0 {
		return fmt.Errorf("missing Pokémon name. Usage: inspect <pokemon-name>")
	}

	pokemonName := strings.ToLower(args[0])
	pokemon, ok := config.CaughtPokemon[pokemonName]
	if !ok {
		return fmt.Errorf("you have not caught that pokemon")
	}

	fmt.Printf("Name: %s\n", pokemon.Name)
	fmt.Printf("Height: %d\n", pokemon.Height)
	fmt.Printf("Weight: %d\n", pokemon.Weight)

	fmt.Println("Stats:")
	for _, stat := range pokemon.Stats {
		fmt.Printf(" -%s: %d\n", stat.Stat.Name, stat.BaseStat)
	}

	fmt.Println("Types:")
	for _, t := range pokemon.Types {
		fmt.Printf(" - %s\n", t.Type.Name)
	}

	return nil
}

func commandExplore(args []string, config *Config) error {
	if len(args) == 0 {
		return fmt.Errorf("missing area name. Usage: explore <area-name>")
	}

	areaName := args[0]
	url, ok := config.Areas[areaName]
	if !ok {
		return fmt.Errorf("area %q not found in current map list. Use the 'map' command to list available areas", areaName)
	}

	fmt.Printf("Exploring %s...\n", areaName)
	return exploreArea(url, config)
}

func exploreArea(url string, config *Config) error {
	if data, ok := config.Cache.Get(url); ok {
		return printPokemonFromArea(data)
	}

	res, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch area data: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read area response body: %w", err)
	}
	if res.StatusCode > 299 {
		return fmt.Errorf("bad status code: %d\nBody: %s", res.StatusCode, body)
	}

	config.Cache.Add(url, body)
	return printPokemonFromArea(body)
}

func printPokemonFromArea(body []byte) error {
	var area struct {
		PokemonEncounters []struct {
			Pokemon struct {
				Name string `json:"name"`
			} `json:"pokemon"`
		} `json:"pokemon_encounters"`
	}

	if err := json.Unmarshal(body, &area); err != nil {
		return fmt.Errorf("failed to parse Pokémon encounters: %w", err)
	}

	if len(area.PokemonEncounters) == 0 {
		fmt.Println("No Pokémon found in this area.")
		return nil
	}

	fmt.Println("Found Pokemon:")
	for _, encounter := range area.PokemonEncounters {
		fmt.Printf(" - %s\n", encounter.Pokemon.Name)
	}
	return nil
}

func showAreas(url string, config *Config, reverse bool) error {

	if data, ok := config.Cache.Get(url); ok {
		return processResponse(data, config, reverse)
	}

	res, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch data: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if res.StatusCode > 299 {
		return fmt.Errorf("bad status code: %d\nBody: %s", res.StatusCode, body)
	}

	if res.StatusCode > 299 {
		return fmt.Errorf("bad status code: %d\nBody: %s", res.StatusCode, body)
	}

	config.Cache.Add(url, body)
	return processResponse(body, config, reverse)
}

func processResponse(body []byte, config *Config, reverse bool) error {
	var areas LocationAreaList
	if err := json.Unmarshal(body, &areas); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Print the location area names
	if reverse {
		for i := len(areas.Results) - 1; i >= 0; i-- {
			fmt.Println(areas.Results[i].Name)
		}
	} else {
		for _, result := range areas.Results {
			fmt.Println(result.Name)
		}
	}

	// Update config with new pagination data
	config.Next = areas.Next
	config.Previous = areas.Previous
	return nil
}

func commandPokedex(args []string, config *Config) error {
	if len(config.CaughtPokemon) == 0 {
		fmt.Println("You haven't caught any Pokémon yet.")
		return nil
	}

	fmt.Println("Your pokedex:")
	for name := range config.CaughtPokemon {
		fmt.Printf(" - %s\n", name)
	}
	return nil
}

func init() {
	commands["help"] = cliCommand{
		name:        "help",
		description: "Displays a help message",
		callback:    commandHelp,
	}

	commands["exit"] = cliCommand{
		name:        "exit",
		description: "Exit the Pokedex",
		callback:    commandExit,
	}

	commands["map"] = cliCommand{
		name:        "map",
		description: "Displays the names of the next 20 location areas in the Pokemon world",
		callback:    commandMap,
	}

	commands["mapb"] = cliCommand{
		name:        "mapb",
		description: "Displays the names of the previous 20 location areas in the Pokemon world",
		callback:    commandMapB,
	}

	commands["explore"] = cliCommand{
		name:        "explore",
		description: "Displays all pokemons in the location area",
		callback:    commandExplore,
	}

	commands["catch"] = cliCommand{
		name:        "catch",
		description: "Attempt to catch a Pokemon",
		callback:    commandCatch,
	}

	commands["inspect"] = cliCommand{
		name:        "inspect",
		description: "It takes the name of a Pokemon and prints the name, height, weight, stats and type(s) of the Pokemon",
		callback:    commandInspect,
	}

	commands["pokedex"] = cliCommand{
		name:        "pokedex",
		description: "It prints a list of all the names of the Pokemon you have caught so far",
		callback:    commandPokedex,
	}

}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	config := &Config{
		Cache:         pokecache.NewCache(5 * time.Second),
		CaughtPokemon: make(map[string]Pokemon),
	}

	if err := loadAllAreas(config); err != nil {
		fmt.Printf("Failed to load areas: %v\n", err)
		return
	}

	for {
		fmt.Printf("Pokedex > ")
		scanner.Scan()
		raw_input := scanner.Text()
		cleaned := cleanInput(raw_input)

		if len(cleaned) == 0 {
			continue // user hit enter without typing anything
		}

		commandName := cleaned[0]

		cmd, exists := commands[commandName]

		if !exists {
			fmt.Println("Unknown command")
			continue
		}

		if err := cmd.callback(cleaned[1:], config); err != nil {
			fmt.Printf("Error executing command %s: %v\n", commandName, err)
		}

	}

}
