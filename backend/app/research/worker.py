async def run_worker(queue, pipeline) -> None:
    while True:
        try:
            job = await queue.dequeue()
            await pipeline.process(job)
        except Exception:
            await asyncio.sleep(0.5)
