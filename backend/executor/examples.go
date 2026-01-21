package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/browserwing/browserwing/services/browser"
)

// ExampleBasicNavigation 示例：基本导航
func ExampleBasicNavigation() error {
	ctx := context.Background()

	// 创建浏览器管理器
	// 注意：在实际使用中，需要提供正确的 config, storage, llmManager 参数
	browserMgr := browser.NewManager(nil, nil, nil)
	if err := browserMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start browser: %w", err)
	}
	defer browserMgr.Stop()

	// 创建 Executor
	executor := NewExecutor(browserMgr)

	// 导航到网页
	result, err := executor.Navigate(ctx, "https://example.com", nil)
	if err != nil {
		return err
	}

	fmt.Printf("Navigation result: %s\n", result.Message)
	return nil
}

// ExampleClickAndType 示例：点击和输入
func ExampleClickAndType() error {
	ctx := context.Background()

	browserMgr := browser.NewManager(nil, nil, nil)
	if err := browserMgr.Start(ctx); err != nil {
		return err
	}
	defer browserMgr.Stop()

	executor := NewExecutor(browserMgr)

	// 导航到搜索页面
	executor.Navigate(ctx, "https://www.google.com", nil)

	// 在搜索框中输入
	result, err := executor.Type(ctx, "input[name='q']", "browserwing", &TypeOptions{
		Clear: true,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Type result: %s\n", result.Message)

	// 点击搜索按钮
	result, err = executor.Click(ctx, "input[name='btnK']", &ClickOptions{
		WaitVisible: true,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Click result: %s\n", result.Message)

	return nil
}

// ExampleSemanticTree 示例：使用语义树
func ExampleSemanticTree() error {
	ctx := context.Background()

	browserMgr := browser.NewManager(nil, nil, nil)
	if err := browserMgr.Start(ctx); err != nil {
		return err
	}
	defer browserMgr.Stop()

	executor := NewExecutor(browserMgr)

	// 导航到页面
	executor.Navigate(ctx, "https://example.com", nil)

	// 获取语义树
	tree, err := executor.GetSemanticTree(ctx)
	if err != nil {
		return err
	}

	// 打印所有可点击元素
	fmt.Println("Clickable Elements (use 'Clickable Element [N]' as identifier):")
	clickable := tree.GetClickableElements()
	for i, node := range clickable {
		fmt.Printf("  [%d] %s (role: %s)\n", i+1, node.Label, node.Role)
	}

	// 打印所有输入元素
	fmt.Println("\nInput Elements (use 'Input Element [N]' as identifier):")
	inputs := tree.GetInputElements()
	for i, node := range inputs {
		fmt.Printf("  [%d] %s (placeholder: %s)\n", i+1, node.Label, node.Placeholder)
	}

	// 推荐方式：使用语义树索引来点击元素
	// 例如：点击第一个可点击元素
	if len(clickable) > 0 {
		executor.Click(ctx, "Clickable Element [1]", nil)
	}

	// 或者：在第一个输入框中输入文本
	if len(inputs) > 0 {
		executor.Type(ctx, "Input Element [1]", "test input", nil)
	}

	return nil
}

// ExampleSmartInteraction 示例：智能交互（通过标签）
func ExampleSmartInteraction() error {
	ctx := context.Background()

	browserMgr := browser.NewManager(nil, nil, nil)
	if err := browserMgr.Start(ctx); err != nil {
		return err
	}
	defer browserMgr.Stop()

	executor := NewExecutor(browserMgr)

	// 导航到登录页面
	executor.Navigate(ctx, "https://example.com/login", nil)

	// 通过标签输入用户名
	result, err := executor.TypeByLabel(ctx, "Username", "myuser")
	if err != nil {
		return err
	}
	fmt.Printf("Username input: %s\n", result.Message)

	// 通过标签输入密码
	result, err = executor.TypeByLabel(ctx, "Password", "mypassword")
	if err != nil {
		return err
	}
	fmt.Printf("Password input: %s\n", result.Message)

	// 通过标签点击登录按钮
	result, err = executor.ClickByLabel(ctx, "Login")
	if err != nil {
		return err
	}
	fmt.Printf("Login button click: %s\n", result.Message)

	return nil
}

// ExampleDataExtraction 示例：数据提取
func ExampleDataExtraction() error {
	ctx := context.Background()

	browserMgr := browser.NewManager(nil, nil, nil)
	if err := browserMgr.Start(ctx); err != nil {
		return err
	}
	defer browserMgr.Stop()

	executor := NewExecutor(browserMgr)

	// 导航到页面
	executor.Navigate(ctx, "https://example.com/products", nil)

	// 提取所有产品标题
	result, err := executor.Extract(ctx, &ExtractOptions{
		Selector: ".product-title",
		Type:     "text",
		Multiple: true,
	})
	if err != nil {
		return err
	}

	// 打印提取的数据
	if data, ok := result.Data["result"].([]map[string]interface{}); ok {
		fmt.Println("Extracted Products:")
		for i, item := range data {
			if text, ok := item["text"].(string); ok {
				fmt.Printf("  [%d] %s\n", i+1, text)
			}
		}
	}

	return nil
}

// ExampleBatchOperations 示例：批量操作
func ExampleBatchOperations() error {
	ctx := context.Background()

	browserMgr := browser.NewManager(nil, nil, nil)
	if err := browserMgr.Start(ctx); err != nil {
		return err
	}
	defer browserMgr.Stop()

	executor := NewExecutor(browserMgr)

	// 定义批量操作
	operations := []Operation{
		{
			Type: "navigate",
			Params: map[string]interface{}{
				"url": "https://example.com/login",
			},
			StopOnError: true,
		},
		{
			Type: "type",
			Params: map[string]interface{}{
				"identifier": "input[name='username']",
				"text":       "myuser",
			},
			StopOnError: true,
		},
		{
			Type: "type",
			Params: map[string]interface{}{
				"identifier": "input[name='password']",
				"text":       "mypassword",
			},
			StopOnError: true,
		},
		{
			Type: "click",
			Params: map[string]interface{}{
				"identifier": "button[type='submit']",
			},
			StopOnError: true,
		},
		{
			Type: "screenshot",
			Params: map[string]interface{}{
				"full_page": true,
			},
			StopOnError: false,
		},
	}

	// 执行批量操作
	result, err := executor.ExecuteBatch(ctx, operations)
	if err != nil {
		return err
	}

	fmt.Printf("Batch execution completed:\n")
	fmt.Printf("  Success: %d\n", result.Success)
	fmt.Printf("  Failed: %d\n", result.Failed)
	fmt.Printf("  Duration: %s\n", result.Duration)

	return nil
}

// ExampleWaitAndScreenshot 示例：等待和截图
func ExampleWaitAndScreenshot() error {
	ctx := context.Background()

	browserMgr := browser.NewManager(nil, nil, nil)
	if err := browserMgr.Start(ctx); err != nil {
		return err
	}
	defer browserMgr.Stop()

	executor := NewExecutor(browserMgr)

	// 导航到页面
	executor.Navigate(ctx, "https://example.com", nil)

	// 等待特定元素出现
	result, err := executor.WaitFor(ctx, ".content-loaded", &WaitForOptions{
		State:   "visible",
		Timeout: 10 * time.Second,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Wait result: %s\n", result.Message)

	// 截图
	result, err = executor.Screenshot(ctx, &ScreenshotOptions{
		FullPage: true,
		Quality:  90,
		Format:   "png",
	})
	if err != nil {
		return err
	}
	fmt.Printf("Screenshot captured: %d bytes\n", result.Data["size"])

	return nil
}

// ExampleFormFilling 示例：表单填写
func ExampleFormFilling() error {
	ctx := context.Background()

	browserMgr := browser.NewManager(nil, nil, nil)
	if err := browserMgr.Start(ctx); err != nil {
		return err
	}
	defer browserMgr.Stop()

	executor := NewExecutor(browserMgr)

	// 导航到表单页面
	executor.Navigate(ctx, "https://example.com/form", nil)

	// 填写文本输入
	executor.Type(ctx, "input[name='name']", "John Doe", nil)
	executor.Type(ctx, "input[name='email']", "john@example.com", nil)

	// 选择下拉框
	executor.Select(ctx, "select[name='country']", "United States", nil)

	// 点击单选按钮
	executor.Click(ctx, "input[name='gender'][value='male']", nil)

	// 点击复选框
	executor.Click(ctx, "input[name='agree']", nil)

	// 提交表单
	result, err := executor.Click(ctx, "button[type='submit']", &ClickOptions{
		WaitVisible: true,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Form submitted: %s\n", result.Message)

	return nil
}

// ExampleAdvancedNavigation 示例：高级导航
func ExampleAdvancedNavigation() error {
	ctx := context.Background()

	browserMgr := browser.NewManager(nil, nil, nil)
	if err := browserMgr.Start(ctx); err != nil {
		return err
	}
	defer browserMgr.Stop()

	executor := NewExecutor(browserMgr)

	// 导航到页面
	executor.Navigate(ctx, "https://example.com/page1", nil)

	// 点击链接导航到另一页
	executor.Click(ctx, "a[href='/page2']", nil)

	// 后退
	result, err := executor.GoBack(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Navigation back: %s\n", result.Message)

	// 前进
	result, err = executor.GoForward(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Navigation forward: %s\n", result.Message)

	// 刷新
	result, err = executor.Reload(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Page reloaded: %s\n", result.Message)

	return nil
}

// ExampleScrolling 示例：滚动操作
func ExampleScrolling() error {
	ctx := context.Background()

	browserMgr := browser.NewManager(nil, nil, nil)
	if err := browserMgr.Start(ctx); err != nil {
		return err
	}
	defer browserMgr.Stop()

	executor := NewExecutor(browserMgr)

	// 导航到页面
	executor.Navigate(ctx, "https://example.com/long-page", nil)

	// 滚动到底部
	result, err := executor.ScrollToBottom(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Scroll result: %s\n", result.Message)

	// 等待加载更多内容
	time.Sleep(2 * time.Second)

	// 滚动到特定元素
	page := executor.GetRodPage()
	if page != nil {
		elem, _ := page.Element("#target-section")
		if elem != nil {
			ScrollToElement(ctx, elem)
		}
	}

	return nil
}
