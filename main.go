package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type cliCommand struct {
	name        string
	description string
	callback    func(*Config) error
}

type Config struct {
	Next     string
	Previous string
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

func commandExit(config *Config) error {
	fmt.Print("Closing the Pokedex... Goodbye!\n")
	os.Exit(0)
	return nil
}

func commandHelp(config *Config) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage: \n")
	for i := range commands {
		cmd := commands[commands[i].name]
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}
	return nil
}

func commandMap(config *Config) error {
	if config.Next == "" {
		config.Next = "https://pokeapi.co/api/v2/location-area/"
	}
	return showAreas(config.Next, config, false)
}

func commandMapB(config *Config) error {
	if config.Previous == "" {
		return fmt.Errorf("no previous page available")
	}
	return showAreas(config.Previous, config, true)
}

func showAreas(url string, config *Config, reverse bool) error {
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
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	config := &Config{} // pagination state

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

		if err := cmd.callback(config); err != nil {
			fmt.Printf("Error executing command %s: %v\n", commandName, err)
		}

	}

}
