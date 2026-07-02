Option Explicit

Function WriteConfig()
    On Error Resume Next
    Dim data, parts, fso, ts, content
    data = Session.Property("CustomActionData")
    parts = Split(data, "|")
    content = "{" & vbCrLf & "  ""url"": ""__GOMSI_SERVERURL__""," & vbCrLf & "  ""token"": ""__GOMSI_TOKEN__""" & vbCrLf & "}" & vbCrLf & ""
    content = Replace(content, "__GOMSI_SERVERURL__", parts(1))
    content = Replace(content, "__GOMSI_TOKEN__", parts(2))
    Set fso = CreateObject("Scripting.FileSystemObject")
    Set ts = fso.CreateTextFile(parts(0), True, False)
    ts.Write content
    ts.Close
    If Err.Number <> 0 Then
        WriteConfig = 3
    Else
        WriteConfig = 1
    End If
End Function
