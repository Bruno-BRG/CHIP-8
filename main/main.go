package main

import "math/rand"

// Chip8 struct
type Chip8 struct {
	memory     [4096]byte    // 4KB
	registers  [16]byte      // V0 to VF
	index      uint16        // index register
	pc         uint16        // program counter
	stack      [16]uint16    // stack
	sp         byte          // stack pointer
	delayTimer byte          // delay timer
	soundTimer byte          // sound timer
	keypad     [16]byte      // 0x0 to 0xF
	display    [64 * 32]byte // 64x32 pixels
	drawFlag   bool          // draw flag
}

// Fonts for the CHIP-8
var fonts = []byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

// NewChip8 creates a new Chip8 instance
func (chip *Chip8) initialize() {
	// Initialize the memory
	for i := 0; i < len(fonts); i++ {
		chip.memory[0x50+i] = fonts[i]
	}
}

func (chip *Chip8) drawSprite(x, y, height byte) {
	chip.registers[0xF] = 0
	for yline := byte(0); yline < height; yline++ {
		pixel := chip.memory[chip.index+uint16(yline)]
		for xline := byte(0); xline < 8; xline++ {
			if (pixel & (0x80 >> xline)) != 0 {
				if chip.display[(int(x)+int(xline)+((int(y)+int(yline))*64))] == 1 {
					chip.registers[0xF] = 1
				}
				chip.display[(int(x) + int(xline) + ((int(y) + int(yline)) * 64))] ^= 1
			}
		}
	}
	chip.drawFlag = true
}

func (chip *Chip8) fetchOpcode() uint16 {
	opcode := uint16(chip.memory[chip.pc])<<8 | uint16(chip.memory[chip.pc+1])
	return opcode
}

func (chip *Chip8) decodeAndExecuteOpcode(opcode uint16) {
	switch opcode & 0xF000 {
	case 0x0000:
		switch opcode & 0x00FF {
		case 0x00E0: // 00E0: Clears the screen
			for i := range chip.display {
				chip.display[i] = 0
			}
			chip.drawFlag = true
		case 0x00EE: // 00EE: Returns from subroutine
			chip.sp--
			chip.pc = chip.stack[chip.sp]
		default:
			panic("Unknown opcode")
		}
	case 0x1000: // 1NNN: Jumps to address NNN
		chip.pc = opcode & 0x0FFF
	case 0x2000: // 2NNN: Calls subroutine at NNN
		chip.stack[chip.sp] = chip.pc
		chip.sp++
		chip.pc = opcode & 0x0FFF
	case 0x3000: // 3XNN: Skips the next instruction if VX equals NN
		if chip.registers[(opcode&0x0F00)>>8] == byte(opcode&0x00FF) {
			chip.pc += 2
		}
	case 0x4000: // 4XNN: Skips the next instruction if VX doesn't equal NN
		if chip.registers[(opcode&0x0F00)>>8] != byte(opcode&0x00FF) {
			chip.pc += 2
		}
	case 0x5000: // 5XY0: Skips the next instruction if VX equals VY
		if chip.registers[(opcode&0x0F00)>>8] == chip.registers[(opcode&0x00F0)>>4] {
			chip.pc += 2
		}
	case 0x6000: // 6XNN: Sets VX to NN
		chip.registers[(opcode&0x0F00)>>8] = byte(opcode & 0x00FF)
	case 0x7000: // 7XNN: Adds NN to VX
		chip.registers[(opcode&0x0F00)>>8] += byte(opcode & 0x00FF)
	case 0x8000:
		x := (opcode & 0x0F00) >> 8
		y := (opcode & 0x00F0) >> 4
		switch opcode & 0x000F {
		case 0x0000: // 8XY0: Sets VX to the value of VY
			chip.registers[x] = chip.registers[y]
		case 0x0001: // 8XY1: Sets VX to VX or VY
			chip.registers[x] |= chip.registers[y]
		case 0x0002: // 8XY2: Sets VX to VX and VY
			chip.registers[x] &= chip.registers[y]
		case 0x0003: // 8XY3: Sets VX to VX xor VY
			chip.registers[x] ^= chip.registers[y]
		case 0x0004: // 8XY4: Adds VY to VX. VF is set to 1 when there's a carry, and to 0 when there isn't
			if chip.registers[y] > (0xFF - chip.registers[x]) {
				chip.registers[0xF] = 1
			} else {
				chip.registers[0xF] = 0
			}
			chip.registers[x] += chip.registers[y]
		case 0x0005: // 8XY5: VY is subtracted from VX. VF is set to 0 when there's a borrow, and 1 when there isn't
			if chip.registers[y] > chip.registers[x] {
				chip.registers[0xF] = 0
			} else {
				chip.registers[0xF] = 1
			}
			chip.registers[x] -= chip.registers[y]
		case 0x0006: // 8XY6: Shifts VX right by one. VF is set to the value of the least significant bit of VX before the shift
			chip.registers[0xF] = chip.registers[x] & 0x1
			chip.registers[x] >>= 1
		case 0x0007: // 8XY7: Sets VX to VY minus VX. VF is set to 0 when there's a borrow, and 1 when there isn't
			if chip.registers[x] > chip.registers[y] {
				chip.registers[0xF] = 0
			} else {
				chip.registers[0xF] = 1
			}
			chip.registers[x] = chip.registers[y] - chip.registers[x]
		case 0x000E: // 8XYE: Shifts VX left by one. VF is set to the value of the most significant bit of VX before the shift
			chip.registers[0xF] = chip.registers[x] >> 7
			chip.registers[x] <<= 1
		default:
			panic("Unknown opcode")
		}
	case 0x9000: // 9XY0: Skips the next instruction if VX doesn't equal VY
		if chip.registers[(opcode&0x0F00)>>8] != chip.registers[(opcode&0x00F0)>>4] {
			chip.pc += 2
		}
	case 0xA000: // ANNN: Sets I to the address NNN
		chip.index = opcode & 0x0FFF
	case 0xB000: // BNNN: Jumps to the address NNN plus V0
		chip.pc = (opcode & 0x0FFF) + uint16(chip.registers[0])
	case 0xC000: // CXNN: Sets VX to a random number and NN
		chip.registers[(opcode&0x0F00)>>8] = byte(rand.Intn(256)) & byte(opcode&0x00FF)
	case 0xD000: // DXYN: Draws a sprite at coordinate (VX, VY) that has a width of 8 pixels and a height of N pixels
		x := chip.registers[(opcode&0x0F00)>>8]
		y := chip.registers[(opcode&0x00F0)>>4]
		height := byte(opcode & 0x000F)
		chip.drawSprite(x, y, height)
	case 0xE000:
		switch opcode & 0x00FF {
		case 0x009E: // EX9E: Skips the next instruction if the key stored in VX is pressed
			if chip.keypad[chip.registers[(opcode&0x0F00)>>8]] != 0 {
				chip.pc += 2
			}
		case 0x00A1: // EXA1: Skips the next instruction if the key stored in VX isn't pressed
			if chip.keypad[chip.registers[(opcode&0x0F00)>>8]] == 0 {
				chip.pc += 2
			}
		default:
			panic("Unknown opcode")
		}
	case 0xF000:
		x := (opcode & 0x0F00) >> 8
		switch opcode & 0x00FF {
		case 0x0007: // FX07: Sets VX to the value of the delay timer
			chip.registers[x] = chip.delayTimer
		case 0x000A: // FX0A: A key press is awaited, and then stored in VX
			keyPress := false
			for i := 0; i < len(chip.keypad); i++ {
				if chip.keypad[i] != 0 {
					chip.registers[x] = byte(i)
					keyPress = true
					break
				}
			}
			if !keyPress {
				return
			}
		case 0x0015: // FX15: Sets the delay timer to VX
			chip.delayTimer = chip.registers[x]
		case 0x0018: // FX18: Sets the sound timer to VX
			chip.soundTimer = chip.registers[x]
		case 0x001E: // FX1E: Adds VX to I
			chip.index += uint16(chip.registers[x])
		case 0x0029: // FX29: Sets I to the location of the sprite for the character in VX
			chip.index = uint16(chip.registers[x]) * 0x5
		case 0x0033: // FX33: Stores the binary-coded decimal representation of VX at the addresses I, I plus 1, and I plus 2
			chip.memory[chip.index] = chip.registers[x] / 100
			chip.memory[chip.index+1] = (chip.registers[x] / 10) % 10
			chip.memory[chip.index+2] = (chip.registers[x] % 100) % 10
		case 0x0055: // FX55: Stores V0 to VX in memory starting at address I
			for i := uint16(0); i <= x; i++ {
				chip.memory[chip.index+i] = chip.registers[i]
			}
		case 0x0065: // FX65: Fills V0 to VX with values from memory starting at address I
			for i := uint16(0); i <= x; i++ {
				chip.registers[i] = chip.memory[chip.index+i]
			}
		default:
			panic("Unknown opcode")
		}
	default:
		panic("Unknown opcode")
	}
}

func main() {
	chip := Chip8{}
	chip.initialize()

	opcode := chip.fetchOpcode()
	chip.decodeAndExecuteOpcode(opcode)
}
