
##### batchqueue vs go channel
1. we can get batch entry from batchqueue, channnel get one by one
2. we can puth batch entry to batchqueue, channnel put one by one
3. batchqueue get entry block when batchqueue is empty, until close batchqueue , same as channel
4. put entry to a closed batchqueue will return a non-nil err, but put entry to a closed channel, it will panic
5. batchqueue also support unblock get. (use batchqueue.TryGet())
6. batchqueue Put api is alway unblock
7. batchqueue PutRoll api can replace oldest data when queue is full, but channel can't 