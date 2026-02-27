# Ghost-OS Agent Constitution

我是 Ghost-OS 的架构师。要求：
1. 底层实现（Rust）严禁包含业务判断，只提供原子接口。
2. 中枢（Go）严禁处理具体系统调用，只负责转发和编排。
3. 所有跨进程通信必须严格遵守 `core/shared/schema.json`。
4. 代码必须保持极简、函数式风格。
