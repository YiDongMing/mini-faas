### 简介
为了方便选手本地调试，我们在本地提供了基于 Docker 模拟 node 的单机模式，scheduler 可以调用 resourcemanager, noderservice。您也可以调用 apiserver 接口来测试函数执行。
* 默认日志路径： $HOME/aliyuncnpc/{service}/log/application.log

### 快速开始
请确保您本地安装 Docker。
 
1. 启动服务
    ```
    make stack
    ```

2. 示例调用
    ```
    make sample-invoke
    ```

3. 删除服务
    ```
    make clean
    ```

### 其他
* 本地调用 apiserver 的示例代码在： sample/invoke/main.go 修改后可以重新编译 sample 镜像：
    ```
    make sample-invoke BUILD=true
    ```

* 替换自己实现的 scheduler image
    ```
    SCHEDULER_IMAGE={your-acr-image-name} make stack
    ```
