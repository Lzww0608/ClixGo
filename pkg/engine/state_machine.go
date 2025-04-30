package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// State 表示状态机中的状态
type State string

// Event 表示触发状态转换的事件
type Event string

// Transition 定义状态转换规则
type Transition struct {
	FromState State
	Event     Event
	ToState   State
	Action    func(ctx context.Context, data interface{}) error
}

// StateMachine 状态机定义
type StateMachine struct {
	currentState State
	transitions  []Transition
	history      []State
	mu           sync.RWMutex
	data         interface{}
	timeout      time.Duration
}

// NewStateMachine 创建新的状态机
func NewStateMachine(initialState State, timeout time.Duration) *StateMachine {
	return &StateMachine{
		currentState: initialState,
		transitions:  make([]Transition, 0),
		history:      []State{initialState},
		timeout:      timeout,
	}
}

// AddTransition 添加状态转换规则
func (sm *StateMachine) AddTransition(from State, event Event, to State, action func(ctx context.Context, data interface{}) error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.transitions = make([]Transition, 0)
	sm.transitions = append(sm.transitions, Transition{
		FromState: from,
		Event:     event,
		ToState:   to,
		Action:    action,
	})
}

// ProcessEvent 处理事件并执行状态转换
func (sm *StateMachine) ProcessEvent(ctx context.Context, event Event, data interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 设置超时上下文
	if sm.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, sm.timeout)
		defer cancel()
	}

	// 查找匹配的转换规则
	for _, transition := range sm.transitions {
		if transition.FromState == sm.currentState && transition.Event == event {
			// 执行转换动作
			if err := transition.Action(ctx, data); err != nil {
				return fmt.Errorf("状态转换失败: %v", err)
			}

			// 更新状态
			sm.currentState = transition.ToState
			sm.history = append(sm.history, transition.ToState)
			sm.data = data

			return nil
		}
	}

	return errors.New("没有找到匹配的状态转换规则")
}

// GetCurrentState 获取当前状态
func (sm *StateMachine) GetCurrentState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// GetHistory 获取状态历史
func (sm *StateMachine) GetHistory() []State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.history
}

// GetData 获取当前状态数据
func (sm *StateMachine) GetData() interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.data
}

// Reset 重置状态机到初始状态
func (sm *StateMachine) Reset(initialState State) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.currentState = initialState
	sm.history = []State{initialState}
	sm.data = nil
}

// Validate 验证状态机定义是否有效
func (sm *StateMachine) Validate() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 检查是否存在循环依赖
	visited := make(map[State]bool)
	recStack := make(map[State]bool)

	var hasCycle func(state State) bool
	hasCycle = func(state State) bool {
		visited[state] = true
		recStack[state] = true

		for _, transition := range sm.transitions {
			if transition.FromState == state {
				if !visited[transition.ToState] {
					if hasCycle(transition.ToState) {
						return true
					}
				} else if recStack[transition.ToState] {
					return true
				}
			}
		}

		recStack[state] = false
		return false
	}

	for _, transition := range sm.transitions {
		if !visited[transition.FromState] {
			if hasCycle(transition.FromState) {
				return errors.New("状态机存在循环依赖")
			}
		}
	}

	return nil
} 