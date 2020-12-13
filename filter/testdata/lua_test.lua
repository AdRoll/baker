function swapFields(rec, next)
    local f1, f2
    f1 = rec:get(1)
    rec:set(1, rec:get(2))
    rec:set(2, f1)
    next(rec)
end

function _fieldByName(rec, next)
    local f1, f2
    f1 = rec:get(fieldByName("bar"))
    rec:set(1, rec:get(fieldByName("baz")))
    rec:set(2, f1)
    next(rec)
end

function _fieldNames(rec, next)
    -- set each field to its name
    rec:set(0, fieldNames[0])
    rec:set(1, fieldNames[1])
    rec:set(2, fieldNames[2])
    next(rec)
end

function _createRecord(rec, next)
    newrec = createRecord()
    newrec:set(0, "hey")
    newrec:set(1, "ho")
    newrec:set(2, "let's go!")
    next(newrec)
    next(rec)
end

function _validateRecord(rec, next)
    ok, idx = validateRecord(rec)
    if ok == false and idx == 0 then
        rec:set(0, "good")
    else
        rec:set(0, "bad")
    end
    next(rec)
end

function clearRecord(rec, next)
    rec:clear()
    next(rec)
end

function copyRecord(rec, next)
    rec:set(2, "1")
    cpy = rec:copy()
    next(rec)

    cpy:set(2, "2")
    next(cpy)
end
