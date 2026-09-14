// Package bindings · 任务生命周期记录（ui-spec §6.3 陈旧锁恢复的 GUI 侧诚实形态）。
// bash 引擎无任何锁机制（flock/lockfile 全仓 grep 核实）——引擎锁查询/半安装清理入口
// 属 v1.1 引擎期，GUI 不假装。可诚实做的：记录 GUI 自身 spawn 的任务生命周期——
// 任务开始写入、完成删除；应用下次启动时文件仍在 = 上次任务被中断
//（应用退出/崩溃/系统断电——bash 事务的回滚只能善后已提交步骤，中断点状态未知）。
// 文件与 offline/.verify-state 同类：bash 既不读也不写，GUI 私有事实，非平行引擎状态。
package bindings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// InterruptedTask 上次中断的任务记录（GetInterruptedTask 返回给前端做启动通知）。
type InterruptedTask struct {
	Cli   string `json:"cli"`   // 完整命令（phpbox mysql install 8.0）
	Stage string `json:"stage"` // 最后推断阶段（configured/preparing/...；无阶段词为空）
	At    string `json:"at"`    // 任务发起时间 RFC3339
}

// taskStateFile 任务生命周期文件（默认 ~/phpbox/.task-state；var 供测试注入隔离）。
var taskStateFile = func() string { return filepath.Join(baseDir(), ".task-state") }

func saveTaskState(cli string) {
	writeTaskState(InterruptedTask{Cli: cli, At: time.Now().Format(time.RFC3339)})
}

// updateTaskStage 阶段推进时回写（中断时能报出"中断于 <阶段>"——§6.3 措辞）。
func updateTaskStage(stage string) {
	if st, err := readTaskState(); err == nil && st.Cli != "" {
		st.Stage = stage
		writeTaskState(st)
	}
}

func clearTaskState() { _ = os.Remove(taskStateFile()) }

func readTaskState() (InterruptedTask, error) {
	raw, err := os.ReadFile(taskStateFile())
	if err != nil {
		return InterruptedTask{}, err
	}
	var st InterruptedTask
	if err := json.Unmarshal(raw, &st); err != nil {
		return InterruptedTask{}, err
	}
	return st, nil
}

func writeTaskState(st InterruptedTask) {
	raw, err := json.Marshal(st)
	if err != nil {
		return
	}
	_ = os.WriteFile(taskStateFile(), raw, 0o644)
}

// GetInterruptedTask 启动期查询上次中断的任务。读取即清除（一次性通知，不反复打扰）；
// 文件损坏视为无中断（清除不报错）。
func (r *Runner) GetInterruptedTask() (InterruptedTask, error) {
	st, err := readTaskState()
	if os.IsNotExist(err) {
		return InterruptedTask{}, nil
	}
	if err != nil {
		clearTaskState()
		return InterruptedTask{}, nil
	}
	clearTaskState()
	return st, nil
}
