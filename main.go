package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gdamore/tcell"
)

var (
	greenStyle = tcell.StyleDefault.Foreground(tcell.ColorGreen)
	heart      = []rune("❤️")
)

const fullBlock = '█'

type loc struct {
	x int
	y int
}

type gameState struct {
	snake []*loc
	heart *loc
	score int

	lock        *sync.Mutex
	heading     direction
	lastHeading direction
	width       int
	height      int
}

func main() {
	screen, err := tcell.NewScreen()
	if err != nil {
		panic(err)
	}

	defer func() {
		err := recover()
		if err != nil {
			panic(err)
			screen.Fini()
		}
	}()

	initError := screen.Init()
	if initError != nil {
		panic(initError)
	}

	screen.Clear()

	width, height := screen.Size()
	height = height - 1
	//-1 since we use the bottom for drawing a status bar
	halfWidth := width / 2
	state := &gameState{
		lock:   &sync.Mutex{},
		width:  width,
		height: height,
		//Let snake start at the bottom and make sure it's not on an odd x coordinate.
		snake:   []*loc{{halfWidth - (halfWidth % 2), height - 1}},
		heading: up,
	}

	screen.SetCell(0, state.height, tcell.StyleDefault, 'S')
	screen.SetCell(1, state.height, tcell.StyleDefault, 'c')
	screen.SetCell(2, state.height, tcell.StyleDefault, 'o')
	screen.SetCell(3, state.height, tcell.StyleDefault, 'r')
	screen.SetCell(4, state.height, tcell.StyleDefault, 'e')
	screen.SetCell(5, state.height, tcell.StyleDefault, ':')
	state.draw(screen)

	//User Eventloop
	go func() {
		for {
			event := screen.PollEvent()
			switch event.(type) {
			case *tcell.EventKey:
				keyEvent := event.(*tcell.EventKey)
				switch keyEvent.Key() {
				case tcell.KeyUp:
					state.changeDirection(up)
				case tcell.KeyRight:
					state.changeDirection(right)
				case tcell.KeyDown:
					state.changeDirection(down)
				case tcell.KeyLeft:
					state.changeDirection(left)
				case tcell.KeyCtrlC:
					screen.Fini()
					os.Exit(0)
				}
			}
		}
	}()

	//Game Updateloop
	gameTicker := time.NewTicker((1000 / 11) * time.Millisecond)
	for {
		<-gameTicker.C
		state.updateSnake(screen)
	}
}

type direction int

const (
	none  direction = 0
	up              = 1
	right           = 2
	down            = 3
	left            = 4
)

func (state *gameState) changeDirection(newDirection direction) {
	state.lock.Lock()
	defer state.lock.Unlock()

	//Directions can only be changed one inbetween one screen-update
	if state.heading == none {
		if len(state.snake) == 1 || state.lastHeading == up && newDirection != down ||
			state.lastHeading == left && newDirection != right ||
			state.lastHeading == right && newDirection != left ||
			state.lastHeading == down && newDirection != up {
			state.heading = newDirection
		}
	}
}

func (state *gameState) gameOver(screen tcell.Screen) {
	screen.Fini()
	fmt.Println("DEAD")
	os.Exit(0)
}

func (state *gameState) updateSnake(screen tcell.Screen) {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.clearScreen(screen)

	tail := state.snake[0]

	var oldHead *loc
	if len(state.snake) == 0 {
		oldHead = tail
	} else {
		oldHead = state.snake[len(state.snake)-1]
	}

	//if the head is out of screen, we are dead
	if oldHead.y < 0 || oldHead.y >= state.height ||
		oldHead.x < 0 || oldHead.x >= state.width {
		state.gameOver(screen)
	}

	var heading direction
	if state.heading == none {
		heading = state.lastHeading
	} else {
		heading = state.heading
	}

	newHead := &loc{oldHead.x, oldHead.y}

	switch heading {
	case up:
		newHead.y = oldHead.y - 1
	case right:
		newHead.x = oldHead.x + 2
	case down:
		newHead.y = oldHead.y + 1
	case left:
		newHead.x = oldHead.x - 2
	}

	for _, bodyPart := range state.snake {
		if bodyPart.x == newHead.x && bodyPart.y == newHead.y {
			state.gameOver(screen)
		}
	}

	grow := false
	if state.heart == nil {
		state.addheart()
	} else {
		if state.heart != nil && newHead.x == state.heart.x && newHead.y == state.heart.y {
			//TODO Check whether snake fills out the field
			grow = true
			state.score = state.score + 1

			state.addheart()
		}
	}

	if !grow {
		state.snake = state.snake[1:]
	}

	state.snake = append(state.snake, newHead)
	state.draw(screen)

	state.lastHeading = heading
	//Resetting the direction to; since user input is ignored if it's
	//not "none". This is in order to avoid two direction changes within
	//a single screen-update.
	state.heading = none
}

// clearScreen only clears the fields of the screen that have already been
// drawn to according to the current state.
func (state *gameState) clearScreen(screen tcell.Screen) {
	if state.heart != nil {
		screen.SetCell(state.heart.x, state.heart.y, tcell.StyleDefault, ' ')
		screen.SetCell(state.heart.x+1, state.heart.y, tcell.StyleDefault, ' ')
	}

	for _, bodyPart := range state.snake {
		screen.SetCell(bodyPart.x, bodyPart.y, tcell.StyleDefault, ' ')
		screen.SetCell(bodyPart.x+1, bodyPart.y, tcell.StyleDefault, ' ')
	}

	//Clear bottombar staring at 7, since we want to leave "Score: "
	for i := 7; i < state.width; i++ {
		screen.SetCell(i, state.height, tcell.StyleDefault, ' ')
	}
}

// draw fills the screen according to state. It draws the heart and the
// snake, followed by pushing the update to the terminal.
func (state *gameState) draw(screen tcell.Screen) {
	if state.heart != nil {
		screen.SetContent(state.heart.x, state.heart.y, heart[0], heart[1:], tcell.StyleDefault)
	}

	for index, bodyPart := range state.snake {
		if index == len(state.snake)-1 {
			screen.SetCell(bodyPart.x, bodyPart.y, greenStyle, fullBlock)
			screen.SetCell(bodyPart.x+1, bodyPart.y, greenStyle, fullBlock)
		} else {
			screen.SetCell(bodyPart.x, bodyPart.y, tcell.StyleDefault, fullBlock)
			screen.SetCell(bodyPart.x+1, bodyPart.y, tcell.StyleDefault, fullBlock)
		}
	}

	//7, since we want to leave a space
	nextCell := 7
	for _, char := range []rune(strconv.Itoa(state.score)) {
		screen.SetCell(nextCell, state.height, tcell.StyleDefault, char)
		nextCell = nextCell + 1
	}

	screen.Show()
}

// addheart sets a new heart for the state. The heart will spawn anywhere,
// except for where any body part of the snake already is.
func (state *gameState) addheart() {
GEN_NEW_heart:
	newheartX, newheartY := generateRandomLocation(state.width, state.height)
	for _, bodyPart := range state.snake {
		if newheartX == bodyPart.x && newheartY == bodyPart.y {
			goto GEN_NEW_heart
		}
	}

	state.heart = &loc{newheartX, newheartY}
}

func generateRandomLocation(width, height int) (int, int) {
	rand.Seed(time.Now().Unix())

	x := rand.Intn(width)
	y := rand.Intn(height)

	if x%2 != 0 {
		x++
	}
	if x >= width-1 {
		x = x - 2
	}
	return x, y
}
