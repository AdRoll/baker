function swapFields(rec)
    local f1, f2 = rec:get(1), rec:get(2)
    rec:set(1, f2)
    rec:set(2, f1)
    return true, nil
end

function errorFromLua(rec)
    return false, "error from lua"
end


function _fieldByName(rec)
    local f1, f2
    f1 = rec:get(fieldByName("bar"))
    rec:set(1, rec:get(fieldByName("baz")))
    rec:set(2, f1)
    return true, nil
end

function _fieldNames(rec)
    -- set each field to its name
    rec:set(0, fieldNames[0])
    rec:set(1, fieldNames[1])
    rec:set(2, fieldNames[2])
    return true, nil
end

function clearRecord(rec)
    rec:clear()
    return true, nil
end