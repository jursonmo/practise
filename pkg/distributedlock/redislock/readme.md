#### redis 分布式锁
1. setnx, 不能强制别人的锁
2. 不能释放别人的锁，所以锁的value是随机的值，释放锁时，必须先判断锁的value一样时，才能释放锁(即删除key)，使用rua 脚本保证原子性
3. 锁要有自动超时功能，避免程序挂后，锁一直不能释放，同时锁的超时时间不宜过长。
4. 锁的超时时间不宜过长，如果在锁的即将超时时，任务还没有完成，可以续约，延迟锁的时间，比如一个任务的deadline是两秒，锁的超时时间就可以500毫秒，如果任务在500ms内完成，就直接释放，如果没有完成，在500ms内完成自动完成续约(延迟锁的超时时间pttl)，以此类推，直到到达任务的deadline的时间点。如果程序挂了，锁最多就锁死500ms, 500ms 后其他app 就可以重新获得锁。
5. 如果锁续约不成功，那么就要通过context 来取消业务层的任务。
6. redis master服务器挂掉，slave 没有同步锁的信息？？向多个master来获取锁，超过一半的master能设置锁，就表示成功。 