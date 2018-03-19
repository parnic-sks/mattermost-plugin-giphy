package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/rpcplugin"
)

// GiphyPlugin is a Mattermost plugin that adds a /gif slash command
// to display a GIF based on user keywords.
type GiphyPlugin struct {
	api           plugin.API
	configuration atomic.Value
	gifProvider   gifProvider
	enabled       bool
	firstLoad     bool
	commands      giphyPluginCommands
}

type giphyPluginCommands struct {
	gifCommand  *model.Command
	gifsCommand *model.Command
}

// OnActivate register the plugin commands
func (p *GiphyPlugin) OnActivate(api plugin.API) error {

	//@Joram Wilander: the plugin crashed after adding two blocks of code : here is the first
	api.RegisterCommand(&model.Command{
		Trigger: "test",
		TeamId:  "",
	})
	// END

	p.api = api
	p.enabled = true

	if err := p.OnConfigurationChange(); err != nil {
		return err
	}

	config := p.config()
	commands := giphyPluginCommands{
		gifCommand:  p.createGifCommand(config.SingleGIFTrigger),
		gifsCommand: p.createGifCommand(config.MultipleGIFTrigger),
	}

	if err := api.RegisterCommand(commands.gifCommand); err != nil {
		return err
	}

	if err := api.RegisterCommand(commands.gifsCommand); err != nil {
		return err
	}
	p.commands = commands

	return nil
}

func (p *GiphyPlugin) config() *GiphyPluginConfiguration {
	config := p.configuration.Load()
	if config != nil {
		return config.(*GiphyPluginConfiguration)
	}
	return nil
}

// OnConfigurationChange handles the changes of plugin configuration
func (p *GiphyPlugin) OnConfigurationChange() error {

	var configuration GiphyPluginConfiguration
	if err := p.api.LoadPluginConfiguration(&configuration); err != nil {
		return err
	}

	if err := configuration.EnsureValidity(); err != nil {
		return err
	}

	if !p.firstLoad {
		if oldConfig := p.config(); oldConfig != nil {
			if oldConfig.SingleGIFTrigger != configuration.SingleGIFTrigger && p.commands.gifCommand != nil {
				if err := p.api.UnregisterCommand("", p.commands.gifCommand.Trigger); err != nil {
					return err
				}
				if err := p.api.RegisterCommand(p.createGifCommand(configuration.SingleGIFTrigger)); err != nil {
					return err
				}
			}
			if oldConfig.MultipleGIFTrigger != configuration.MultipleGIFTrigger && p.commands.gifsCommand != nil {
				if err := p.api.UnregisterCommand("", p.commands.gifsCommand.Trigger); err != nil {
					return err
				}
				if err := p.api.RegisterCommand(p.createGifCommand(configuration.MultipleGIFTrigger)); err != nil {
					return err
				}
			}
		}
	}
	p.configuration.Store(&configuration)
	p.firstLoad = false
	return nil
}

// OnDeactivate handles plugin deactivation
func (p *GiphyPlugin) OnDeactivate() error {
	p.enabled = false
	return nil
}

func (p *GiphyPlugin) createGifCommand(trigger string) *model.Command {
	return &model.Command{
		Trigger:          trigger,
		Description:      "Posts a Giphy GIF that matches the keyword(s)",
		DisplayName:      "Giphy command",
		AutoComplete:     true,
		AutoCompleteDesc: "Posts a Giphy GIF that matches the keyword(s)",
		AutoCompleteHint: "happy kitty",
	}
}

func (p *GiphyPlugin) createGifsCommand(trigger string) *model.Command {
	return &model.Command{
		Trigger:          trigger,
		Description:      "Shows a preview of 10 GIFS matching the keyword(s)",
		DisplayName:      "Giphy preview command",
		AutoComplete:     true,
		AutoCompleteDesc: "Shows a preview of 10 GIFS matching the keyword(s)",
		AutoCompleteHint: "happy kitty",
	}
}

// ExecuteCommand returns a post that displays a GIF choosen using Giphy
func (p *GiphyPlugin) ExecuteCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	//@Joram Wilander: the plugin crashed after adding two blocks of code : here is the second
	// (trying to make unregistering work because I haven't yet managed to)
	if strings.HasPrefix(args.Command, "/test") {
		p.api.UnregisterCommand("", "test")
	}
	//END

	if !p.enabled {
		return nil, appError("Cannot execute command while the plugin is disabled.", nil)
	}
	if p.api == nil {
		return nil, appError("Cannot access the plugin API.", nil)
	}
	config := p.config()
	if strings.HasPrefix(args.Command, "/"+config.MultipleGIFTrigger) {
		return p.executeCommandGifs(args.Command)
	}
	if strings.HasPrefix(args.Command, "/"+config.SingleGIFTrigger) {
		return p.executeCommandGif(args.Command)
	}

	return nil, appError("Command trigger "+args.Command+" is not supported by this plugin.", nil)
}

// executeCommandGif returns a public post containing a matching GIF
func (p *GiphyPlugin) executeCommandGif(command string) (*model.CommandResponse, *model.AppError) {
	config := p.config()
	keywords := getCommandKeywords(command, config.SingleGIFTrigger)
	gifURL, err := p.gifProvider.getGifURL(p.config(), keywords)
	if err != nil {
		return nil, appError("Unable to get GIF URL", err)
	}

	text := " *[" + keywords + "](" + gifURL + ")*\n" + "![GIF for '" + keywords + "'](" + gifURL + ")"
	return &model.CommandResponse{ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, Text: text}, nil
}

// executeCommandGif returns a private post containing a list of matching GIFs
func (p *GiphyPlugin) executeCommandGifs(command string) (*model.CommandResponse, *model.AppError) {
	config := p.config()
	keywords := getCommandKeywords(command, config.MultipleGIFTrigger)
	gifURLs, err := p.gifProvider.getMultipleGifsURL(p.config(), keywords)
	if err != nil {
		return nil, appError("Unable to get GIF URL", err)
	}

	text := fmt.Sprintf(" *Suggestions for '%s':*\n", keywords)
	for i, url := range gifURLs {
		if i > 0 {
			text += "\t"
		}
		text += fmt.Sprintf("[![GIF for '%s'](%s)](%s)", keywords, url, url)
	}
	return &model.CommandResponse{ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL, Text: text}, nil
}

func getCommandKeywords(commandLine string, trigger string) string {
	return strings.Replace(commandLine, "/"+trigger, "", 1)
}

func appError(message string, err error) *model.AppError {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}
	return model.NewAppError("Giphy Plugin", message, nil, errorMessage, http.StatusBadRequest)
}

// Install the RCP plugin
func main() {
	plugin := GiphyPlugin{}
	plugin.gifProvider = &giphyProvider{}
	plugin.firstLoad = true
	rpcplugin.Main(&plugin)
}
