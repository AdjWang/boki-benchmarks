def invoke(callee, input):
    tagPre = hashLogTag([ID, STEP, "pre"])
    logAppend(tags: [tagPre],
              data: {"calleeId": UUID()})
    rec = logReadNext(tag: tagPre, minSeqnum: 0)
    calleeId = rec.data["calleeId"]
    retVal = rawInvoke(callee, [calleeId, input])
    tagPost = hashLogTag([ID, STEP, "post"])
    logAppend(tags: [tagPost],
              data: {"retVal": retVal})
    rec = logReadNext(tag: tagPost, minSeqnum: 0)
    STEP = STEP + 1
    return rec.data["retVal"]


def invoke(callee, input):
    STEP = STEP + 1
    tagInvoke = hashLogTag([ID, STEP, "pre"])
    logAppend(tags: [tagInvoke],
              data: {"calleeId": UUID()})
    rec = logReadNext(tag: tagInvoke, minSeqnum: 0)
    if rec != None:
        return rec.data["retVal"]

    calleeId = UUID()
    ctx = Context(NewContext(), calleeId, input)
    retVal = rawInvoke(ctx, callee)

    return retVal





def write(ctx, table, key, val):
    STEP = STEP + 1
    tag = hashLogTag([ID, STEP, "pre"])
    lastStepPre = asyncLogAppend(tags: [tag],
                                 data: [table, key, val],
                                 dep: ctx.logs.lastStep,
                                 cond: Cond_IsTheFirstStep)
    ctx.logs.chain(lastStepPre)
    # await and resolve dependencies and conditions
    ctx.logs.sync()
    rec = logReadNext(tag: tag, minSeqnum: 0)
    rawDBWrite(table, key,
               cond: "Version < {rec.seqnum}",
               update: "Value={val}; Version={rec.seqnum}")
    tag = hashLogTag([ID, STEP, "post"])
    lastStepPost = asyncLogAppend(tags: [tag],
                                  data: [table, key, val],
                                  dep: lastStepPre)
    ctx.logs.chain(lastStepPost)


def invoke(ctx, callee, input):
    STEP = STEP + 1
    tag = hashLogTag([ID, STEP])
    calleeId = UUID()
    lastStep = asyncLogAppend(tags: [tag],
                              data: {"calleeId": calleeId},
                              dep: ctx.logs.lastStep,
                              cond: Cond_IsTheFirstStep)
    ctx.logs.chain(lastStep)
    calleeId = rec.data["calleeId"]
    retVal = rawInvoke(ctx, callee, [ctx, calleeId, input])
    return retVal
