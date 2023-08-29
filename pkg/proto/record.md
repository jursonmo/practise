1. 2023-08-27, 增加client server 库， example5 目录下有使用例子。
    有心跳机制，业务层用注册路由AddRouter() 

2. 2023-08-27,业务层发送数据时，需要制定msgid. 
   TODO: 数据在发送过程中，make 内存对象次数有点多，应该在proto 层面实现指定msgid.
   2023-08-29, msgid 在proto option层面实现了，但是构建option时还是make 一个小对象了，相比之前make 一个大对象好一点而已