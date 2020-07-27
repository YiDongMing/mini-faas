### 简介
为了方便选手本地调试，我们在本地提供了基于 Docker 模拟 node 的单机模式，scheduler 可以调用 resourcemanager, noderservice。您也可以调用 apiserver 接口来测试函数执行。
* 默认日志路径： $HOME/aliyuncnpc/{service}/log/application.log

### 快速开始
请确保您本地安装 Docker。
 
* 使用提供的已构建好的 scheduler demo 镜像启动服务
    ```
    make stack
    ```

* 或者构建本地 scheduler 镜像，并启动服务
    ```
    make stack BUILD=true SCHEDULER_IMAGE=registry.cn-hangzhou.aliyuncs.com/{namespace}/{image}:{tag}
    ```

* 示例调用
    ```
    make sample-invoke
    ```

* 删除服务
    ```
    make clean
    ```
* 发布镜像
    ```
    docker push registry.cn-hangzhou.aliyuncs.com/{namespace}/{image}:{tag}
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
