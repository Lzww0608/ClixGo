package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewStateMachine(t *testing.T) {
	initialState := State("初始状态")
	timeout := 5 * time.Second

	sm := NewStateMachine(initialState, timeout)

	if sm.currentState != initialState {
		t.Errorf("期望初始状态为 %s, 得到 %s", initialState, sm.currentState)
	}

	if sm.timeout != timeout {
		t.Errorf("期望超时为 %v, 得到 %v", timeout, sm.timeout)
	}

	if len(sm.history) != 1 || sm.history[0] != initialState {
		t.Errorf("期望历史记录为 [%s], 得到 %v", initialState, sm.history)
	}
}

func TestAddTransition(t *testing.T) {
	sm := NewStateMachine("初始状态", 0)

	fromState := State("状态A")
	event := Event("事件1")
	toState := State("状态B")

	action := func(ctx context.Context, data interface{}) error {
		return nil
	}

	sm.AddTransition(fromState, event, toState, action)

	if len(sm.transitions) != 1 {
		t.Fatalf("期望转换规则数量为 1, 得到 %d", len(sm.transitions))
	}

	transition := sm.transitions[0]
	if transition.FromState != fromState {
		t.Errorf("期望FromState为 %s, 得到 %s", fromState, transition.FromState)
	}
	if transition.Event != event {
		t.Errorf("期望Event为 %s, 得到 %s", event, transition.Event)
	}
	if transition.ToState != toState {
		t.Errorf("期望ToState为 %s, 得到 %s", toState, transition.ToState)
	}

	// 测试添加多个转换
	sm.AddTransition(State("状态B"), Event("事件2"), State("状态C"), action)

	// 现在我们应该有2个转换规则
	if len(sm.transitions) != 2 {
		t.Fatalf("期望转换规则数量为 2, 得到 %d", len(sm.transitions))
	}

	// 确认第二个转换规则正确添加
	secondTransition := sm.transitions[1]
	if secondTransition.FromState != "状态B" {
		t.Errorf("期望第二个转换的FromState为 状态B, 得到 %s", secondTransition.FromState)
	}
	if secondTransition.Event != "事件2" {
		t.Errorf("期望第二个转换的Event为 事件2, 得到 %s", secondTransition.Event)
	}
	if secondTransition.ToState != "状态C" {
		t.Errorf("期望第二个转换的ToState为 状态C, 得到 %s", secondTransition.ToState)
	}
}

func TestProcessEvent(t *testing.T) {
	sm := NewStateMachine("初始状态", 0)

	actionCalled := false
	action := func(ctx context.Context, data interface{}) error {
		actionCalled = true
		return nil
	}

	sm.currentState = "状态A"
	sm.transitions = []Transition{{
		FromState: "状态A",
		Event:     "事件1",
		ToState:   "状态B",
		Action:    action,
	}}

	err := sm.ProcessEvent(context.Background(), "事件1", "测试数据")
	if err != nil {
		t.Errorf("处理事件应该成功，得到错误: %v", err)
	}
	if !actionCalled {
		t.Error("Action应该被调用")
	}
	if sm.currentState != "状态B" {
		t.Errorf("期望当前状态为 状态B, 得到 %s", sm.currentState)
	}
	if sm.data != "测试数据" {
		t.Errorf("期望数据为 测试数据, 得到 %v", sm.data)
	}
}

func TestProcessEvent_NoMatchingTransition(t *testing.T) {
	sm := NewStateMachine("初始状态", 0)

	err := sm.ProcessEvent(context.Background(), "不存在的事件", nil)
	if err == nil {
		t.Error("处理不存在的事件应该返回错误")
	}
}

func TestProcessEvent_ActionError(t *testing.T) {
	sm := NewStateMachine("初始状态", 0)

	expectedErr := errors.New("动作执行失败")
	failingAction := func(ctx context.Context, data interface{}) error {
		return expectedErr
	}

	sm.currentState = "状态A"
	sm.transitions = []Transition{{
		FromState: "状态A",
		Event:     "事件1",
		ToState:   "状态B",
		Action:    failingAction,
	}}

	err := sm.ProcessEvent(context.Background(), "事件1", nil)
	if err == nil {
		t.Error("处理失败的动作应该返回错误")
	}
}

func TestProcessEvent_Timeout(t *testing.T) {
	// 设置一个非常短的超时
	sm := NewStateMachine("初始状态", 1*time.Millisecond)

	timeoutAction := func(ctx context.Context, data interface{}) error {
		// 模拟超时操作，休眠比超时设置更长的时间
		select {
		case <-time.After(10 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	sm.currentState = "状态A"
	sm.transitions = []Transition{{
		FromState: "状态A",
		Event:     "事件1",
		ToState:   "状态B",
		Action:    timeoutAction,
	}}

	err := sm.ProcessEvent(context.Background(), "事件1", nil)
	if err == nil {
		t.Error("超时操作应该返回错误")
	}
}

func TestGetCurrentState(t *testing.T) {
	expectedState := State("测试状态")
	sm := NewStateMachine(expectedState, 0)

	if sm.GetCurrentState() != expectedState {
		t.Errorf("期望当前状态为 %s, 得到 %s", expectedState, sm.GetCurrentState())
	}
}

func TestGetHistory(t *testing.T) {
	initialState := State("初始状态")
	sm := NewStateMachine(initialState, 0)

	history := sm.GetHistory()
	if len(history) != 1 || history[0] != initialState {
		t.Errorf("期望历史记录为 [%s], 得到 %v", initialState, history)
	}
}

func TestGetData(t *testing.T) {
	sm := NewStateMachine("初始状态", 0)

	expectedData := "测试数据"
	sm.data = expectedData

	if sm.GetData() != expectedData {
		t.Errorf("期望数据为 %v, 得到 %v", expectedData, sm.GetData())
	}
}

func TestReset(t *testing.T) {
	sm := NewStateMachine("初始状态", 0)

	// 先更改状态，添加历史记录和数据
	sm.currentState = "中间状态"
	sm.history = append(sm.history, "中间状态")
	sm.data = "一些数据"

	// 重置
	newInitialState := State("新初始状态")
	sm.Reset(newInitialState)

	if sm.currentState != newInitialState {
		t.Errorf("期望当前状态为 %s, 得到 %s", newInitialState, sm.currentState)
	}

	if len(sm.history) != 1 || sm.history[0] != newInitialState {
		t.Errorf("期望历史记录为 [%s], 得到 %v", newInitialState, sm.history)
	}

	if sm.data != nil {
		t.Errorf("期望数据为 nil, 得到 %v", sm.data)
	}
}

func TestValidate_NoCycle(t *testing.T) {
	sm := NewStateMachine("状态A", 0)

	// 创建无循环的状态转换
	sm.transitions = []Transition{
		{FromState: "状态A", Event: "事件1", ToState: "状态B"},
		{FromState: "状态B", Event: "事件2", ToState: "状态C"},
		{FromState: "状态C", Event: "事件3", ToState: "状态D"},
	}

	err := sm.Validate()
	if err != nil {
		t.Errorf("无循环的状态机应该通过验证，得到错误: %v", err)
	}
}

func TestValidate_WithCycle(t *testing.T) {
	sm := NewStateMachine("状态A", 0)

	// 创建有循环的状态转换
	sm.transitions = []Transition{
		{FromState: "状态A", Event: "事件1", ToState: "状态B"},
		{FromState: "状态B", Event: "事件2", ToState: "状态C"},
		{FromState: "状态C", Event: "事件3", ToState: "状态A"}, // 形成循环
	}

	err := sm.Validate()
	if err == nil {
		t.Error("有循环的状态机应该返回错误")
	}
}
