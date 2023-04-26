def invoke(callee, input):
    tagPre = hashLogTag([ID, STEP, "pre"])
    logAppend(tags: [tagPre],
              data: { "calleeId": UUID() })
    rec = logReadNext(tag: tagPre, minSeqnum: 0)
    calleeId = rec.data["calleeId"]
    retVal = rawInvoke(callee, [calleeId, input])
    tagPost = hashLogTag([ID, STEP, "post"])
    logAppend(tags: [tagPost],
              data: { "retVal": retVal })
    rec = logReadNext(tag: tagPost, minSeqnum: 0)
    STEP = STEP + 1
    return rec.data["retVal"]

def invoke(callee, input):
    STEP = STEP + 1
    tagInvoke = hashLogTag([ID, STEP, "pre"])
    logAppend(tags: [tagInvoke],
              data: { "calleeId": UUID() })
    rec = logReadNext(tag: tagInvoke, minSeqnum: 0)
    if rec != None:
        return rec.data["retVal"]

    calleeId = UUID()
    ctx = Context(NewContext(), calleeId, input)
    retVal = rawInvoke(ctx, callee)

    return retVal

def rawInvokeWrapper(ctx, user_func):
    logAppend(tags: [IntentLogTag],
              data: { "instanceId": ctx.callee.ID })

    retVal = user_func(ctx.callee.Input)

    if ctx.caller != None:
        tagInvoke = hashLogTag([ctx.caller.ID, ctx.caller.STEP, "callback"])
        logAppend(tags: [tagInvoke]
                  data: { "result": retVal })

    logAppend(tags: [IntentLogTag],
              data: { "Done": True })

    return retVal

def invoke(callee, input):
    STEP = STEP + 1
    tagInvoke = hashLogTag([ID, STEP, "pre"])
    lastStep = asyncLogAppend(tags: [tagInvoke],
                              data: { "calleeId": UUID() })
               .chain(asyncLogReadNext(tag: tagInvoke, minSeqnum: 0))

    calleeId = UUID()
    ctx = Context(NewContext(), calleeId, input, lastStep)
    retVal = rawInvoke(ctx, callee)

    return retVal

def rawInvokeWrapper(ctx, user_func):
    lastStep = ctx.lastStep
    lastStep.chain(asyncLogAppend(tags: [IntentLogTag],
                                  data: { "instanceId": ctx.calleeID }))
    if await lastStep.verify():
        return lastStep.unwrap().data["retVal"]

    retVal = user_func(ctx.calleeInput)

    if ctx.caller != None:
        tagInvoke = hashLogTag([ctx.callerID, ctx.callerSTEP, "callback"])
        lastStep.chain(asyncLogAppend(tags: [tagInvoke]
                                      data: { "result": retVal }))

    lastStep.chain(asyncLogAppend(tags: [IntentLogTag],
                                  data: { "Done": True }))
    ctx.Update(lastStep)

    return retVal
