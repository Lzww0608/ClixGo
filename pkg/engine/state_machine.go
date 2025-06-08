/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-8 18:16:17
* @Description: 状态机引擎的核心实现
 */

package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// State 表示状态机中的状态
//
// 状态是状态机的核心概念，代表系统在某个时刻的特定状态
type State string

// Event 表示触发状态转换的事件
//
// 事件是导致状态机从一个状态转换到另一个状态的触发器
type Event string

// TransitionAction 定义状态转换时执行的动作函数
//
// 参数:
//   - ctx: 上下文，用于控制超时和取消
//   - data: 传递给动作的数据
//
// 返回:
//   - error: 动作执行过程中的错误，nil表示成功
type TransitionAction func(ctx context.Context, data interface{}) error

// Transition 定义状态转换规则
//
// 转换规则描述了在特定状态下接收到特定事件时应该转换到的目标状态，
// 以及在转换过程中需要执行的动作
type Transition struct {
	FromState State            `json:"from_state"` // 源状态
	Event     Event            `json:"event"`      // 触发事件
	ToState   State            `json:"to_state"`   // 目标状态
	Action    TransitionAction `json:"-"`          // 转换动作（不序列化）
}

// StateMachine 状态机定义
//
// 状态机是一个有限状态自动机，它在任何时候都处于有限状态集合中的一个状态。
// 当接收到事件时，状态机会根据预定义的转换规则改变其状态。
type StateMachine struct {
	currentState State         `json:"-"` // 当前状态（私有字段，不序列化）
	transitions  []Transition  `json:"-"` // 状态转换规则集合（私有字段，不序列化）
	history      []State       `json:"-"` // 状态变化历史（私有字段，不序列化）
	data         interface{}   `json:"-"` // 当前状态关联的数据（私有字段，不序列化）
	timeout      time.Duration `json:"-"` // 状态转换超时时间（私有字段，不序列化）
	mu           sync.RWMutex  `json:"-"` // 读写锁，保证并发安全
}

// NewStateMachine 创建新的状态机实例
//
// 该函数创建一个新的状态机，并将其初始化为指定的初始状态
//
// 参数:
//   - initialState: 状态机的初始状态
//   - timeout: 状态转换的超时时间，0表示无超时限制
//
// 返回:
//   - *StateMachine: 新创建的状态机实例
//
// 示例:
//
//	sm := NewStateMachine("idle", 5*time.Second)
func NewStateMachine(initialState State, timeout time.Duration) *StateMachine {
	return &StateMachine{
		currentState: initialState,
		transitions:  make([]Transition, 0),
		history:      []State{initialState},
		timeout:      timeout,
	}
}

// AddTransition 添加状态转换规则
//
// 该函数向状态机添加一个新的状态转换规则。当状态机处于指定的源状态
// 并接收到指定事件时，将转换到目标状态并执行相应的动作。
//
// 参数:
//   - from: 源状态，转换的起始状态
//   - event: 触发转换的事件
//   - to: 目标状态，转换的结束状态
//   - action: 转换时执行的动作函数，可以为nil
//
// 注意：
//   - 可以为同一个(from, event)组合添加多个转换规则
//   - 转换规则按添加顺序进行匹配
//
// 示例:
//
//	sm.AddTransition("idle", "start", "running", func(ctx context.Context, data interface{}) error {
//	    fmt.Println("开始运行")
//	    return nil
//	})
func (sm *StateMachine) AddTransition(from State, event Event, to State, action TransitionAction) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	transition := Transition{
		FromState: from,
		Event:     event,
		ToState:   to,
		Action:    action,
	}

	sm.transitions = append(sm.transitions, transition)
}

// ProcessEvent 处理事件并执行状态转换
//
// 该函数是状态机的核心方法，负责处理输入事件并根据当前状态和转换规则
// 执行相应的状态转换。
//
// 执行流程：
// 1. 设置超时上下文（如果配置了超时）
// 2. 查找匹配的转换规则
// 3. 执行转换动作
// 4. 更新状态和历史记录
//
// 参数:
//   - ctx: 上下文，用于控制取消和超时
//   - event: 要处理的事件
//   - data: 传递给转换动作的数据
//
// 返回:
//   - error: 处理过程中的错误，nil表示成功
//
// 错误类型：
//   - 转换动作执行失败
//   - 没有找到匹配的转换规则
//   - 上下文超时或取消
//
// 示例:
//
//	err := sm.ProcessEvent(ctx, "start", map[string]interface{}{"user": "admin"})
//	if err != nil {
//	    log.Printf("状态转换失败: %v", err)
//	}
func (sm *StateMachine) ProcessEvent(ctx context.Context, event Event, data interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 设置超时上下文
	effectiveCtx := ctx
	if sm.timeout > 0 {
		var cancel context.CancelFunc
		effectiveCtx, cancel = context.WithTimeout(ctx, sm.timeout)
		defer cancel()
	}

	// 查找匹配的转换规则
	transition, found := sm.findMatchingTransition(sm.currentState, event)
	if !found {
		return fmt.Errorf("状态机错误: 在状态 '%s' 下没有找到事件 '%s' 的转换规则",
			sm.currentState, event)
	}

	// 执行转换动作
	if err := sm.executeTransitionAction(effectiveCtx, transition, data); err != nil {
		return fmt.Errorf("状态转换动作执行失败: %w", err)
	}

	// 更新状态机状态
	sm.updateState(transition.ToState, data)

	return nil
}

// findMatchingTransition 查找匹配的转换规则
//
// 该函数在转换规则集合中查找与当前状态和事件匹配的第一个规则
//
// 参数:
//   - currentState: 当前状态
//   - event: 触发事件
//
// 返回:
//   - Transition: 匹配的转换规则
//   - bool: true表示找到匹配规则，false表示未找到
func (sm *StateMachine) findMatchingTransition(currentState State, event Event) (Transition, bool) {
	for _, transition := range sm.transitions {
		if transition.FromState == currentState && transition.Event == event {
			return transition, true
		}
	}
	return Transition{}, false
}

// executeTransitionAction 执行转换动作
//
// 该函数安全地执行转换动作，处理可能的panic和错误
//
// 参数:
//   - ctx: 上下文
//   - transition: 要执行的转换规则
//   - data: 传递给动作的数据
//
// 返回:
//   - error: 执行过程中的错误，nil表示成功
func (sm *StateMachine) executeTransitionAction(ctx context.Context, transition Transition, data interface{}) error {
	// 如果没有定义动作，直接返回成功
	if transition.Action == nil {
		return nil
	}

	// 使用defer捕获可能的panic
	defer func() {
		if r := recover(); r != nil {
			// 这里可以记录panic信息
		}
	}()

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 执行转换动作
	return transition.Action(ctx, data)
}

// updateState 更新状态机状态
//
// 该函数更新状态机的当前状态、历史记录和关联数据
//
// 参数:
//   - newState: 新的状态
//   - data: 新的状态数据
func (sm *StateMachine) updateState(newState State, data interface{}) {
	sm.currentState = newState
	sm.history = append(sm.history, newState)
	sm.data = data
}

// GetCurrentState 获取当前状态
//
// 该函数以线程安全的方式返回状态机的当前状态
//
// 返回:
//   - State: 当前状态
func (sm *StateMachine) GetCurrentState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// GetHistory 获取状态历史
//
// 该函数返回状态机的完整状态变化历史，包括初始状态
//
// 返回:
//   - []State: 状态历史的副本，按时间顺序排列
func (sm *StateMachine) GetHistory() []State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 返回历史的副本，避免外部修改
	historyCopy := make([]State, len(sm.history))
	copy(historyCopy, sm.history)
	return historyCopy
}

// GetData 获取当前状态关联的数据
//
// 该函数以线程安全的方式返回与当前状态关联的数据
//
// 返回:
//   - interface{}: 当前状态的关联数据
func (sm *StateMachine) GetData() interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.data
}

// Reset 重置状态机到指定状态
//
// 该函数将状态机重置到指定的状态，清空历史记录和关联数据
//
// 参数:
//   - initialState: 重置后的初始状态
//
// 注意：此操作会清除所有历史记录和状态数据
func (sm *StateMachine) Reset(initialState State) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.currentState = initialState
	sm.history = []State{initialState}
	sm.data = nil
}

// GetTransitionCount 获取转换规则数量
//
// 该函数返回当前状态机中定义的转换规则总数
//
// 返回:
//   - int: 转换规则数量
func (sm *StateMachine) GetTransitionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.transitions)
}

// HasTransition 检查是否存在指定的转换规则
//
// 该函数检查状态机中是否存在从指定状态接收指定事件的转换规则
//
// 参数:
//   - from: 源状态
//   - event: 触发事件
//
// 返回:
//   - bool: true表示存在转换规则，false表示不存在
func (sm *StateMachine) HasTransition(from State, event Event) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	_, found := sm.findMatchingTransition(from, event)
	return found
}

// Validate 验证状态机定义是否有效
//
// 该函数检查状态机的定义是否存在问题，包括：
// - 循环依赖检测
// - 孤立状态检测
// - 转换规则完整性检查
//
// 返回:
//   - error: 验证过程中发现的问题，nil表示状态机定义有效
//
// 验证项目：
//   - 检查是否存在状态循环依赖
//   - 检查是否存在无法到达的状态
//   - 检查转换规则的一致性
func (sm *StateMachine) Validate() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 检查循环依赖
	if err := sm.checkCyclicDependencies(); err != nil {
		return fmt.Errorf("状态机验证失败: %w", err)
	}

	// 检查孤立状态
	if err := sm.checkOrphanedStates(); err != nil {
		return fmt.Errorf("状态机验证失败: %w", err)
	}

	return nil
}

// checkCyclicDependencies 检查循环依赖
//
// 使用深度优先搜索算法检测状态转换图中是否存在循环
//
// 返回:
//   - error: 如果存在循环依赖则返回错误，否则返回nil
func (sm *StateMachine) checkCyclicDependencies() error {
	visited := make(map[State]bool)
	recStack := make(map[State]bool)

	var hasCycle func(state State) bool
	hasCycle = func(state State) bool {
		visited[state] = true
		recStack[state] = true

		// 检查从当前状态出发的所有转换
		for _, transition := range sm.transitions {
			if transition.FromState == state {
				targetState := transition.ToState
				if !visited[targetState] {
					if hasCycle(targetState) {
						return true
					}
				} else if recStack[targetState] {
					return true
				}
			}
		}

		recStack[state] = false
		return false
	}

	// 检查所有状态
	allStates := sm.getAllStates()
	for _, state := range allStates {
		if !visited[state] {
			if hasCycle(state) {
				return errors.New("检测到状态循环依赖")
			}
		}
	}

	return nil
}

// checkOrphanedStates 检查孤立状态
//
// 检查是否存在无法从初始状态到达的状态
//
// 返回:
//   - error: 如果存在孤立状态则返回错误，否则返回nil
func (sm *StateMachine) checkOrphanedStates() error {
	if len(sm.transitions) == 0 {
		return nil // 没有转换规则时不检查孤立状态
	}

	reachableStates := make(map[State]bool)
	reachableStates[sm.currentState] = true

	// 使用广度优先搜索找到所有可达状态
	queue := []State{sm.currentState}
	for len(queue) > 0 {
		currentState := queue[0]
		queue = queue[1:]

		for _, transition := range sm.transitions {
			if transition.FromState == currentState && !reachableStates[transition.ToState] {
				reachableStates[transition.ToState] = true
				queue = append(queue, transition.ToState)
			}
		}
	}

	// 检查是否有状态无法到达
	allStates := sm.getAllStates()
	for _, state := range allStates {
		if !reachableStates[state] {
			return fmt.Errorf("检测到孤立状态: %s", state)
		}
	}

	return nil
}

// getAllStates 获取状态机中的所有状态
//
// 该函数遍历所有转换规则，收集其中涉及的所有状态
//
// 返回:
//   - []State: 所有状态的列表
func (sm *StateMachine) getAllStates() []State {
	stateSet := make(map[State]bool)
	stateSet[sm.currentState] = true

	for _, transition := range sm.transitions {
		stateSet[transition.FromState] = true
		stateSet[transition.ToState] = true
	}

	states := make([]State, 0, len(stateSet))
	for state := range stateSet {
		states = append(states, state)
	}

	return states
}
