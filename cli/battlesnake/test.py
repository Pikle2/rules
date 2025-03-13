import os
import tkinter
import webview
#os.system('go run main.go')
testGui = tkinter.Tk(screenName=None,baseName=None,className='BattleSnake Run',useTk=1)
snakeName = tkinter.StringVar()
snakePort = tkinter.IntVar()
testGui.attributes('-topmost', True)
#snake1 = "soup"
#snake2 = "the_thunker"
snakelist = []
snakesRun = {}
snakeDict = {}
i = 0
check = []
kill = []
names = []
def addsnake(texts,ports: int):
    global i
    global snakePort
    global snakeName
    try:
        texts = inputsnake.get()
        ports = int(inputport.get())
        #if isinstance(ports, int):
            #print("test")
        ports2 = str(ports)
        if len(ports2) == 4 and len(texts) >= 1 and len(texts) <= 20 and len(snakelist) <= 25:
            snakelist.append(texts)
            snakeDict[texts] = "http://0.0.0.0:" + ports2
            snakesRun[texts] = False
            tempButton = tkinter.Button(testGui, text = "\u2610", command = lambda j=i: testcheck(j))
            tempButton.place(x=0, y=(25*i) + 50)
            tempButton.config(bd=0)
            check.append(tempButton)
            labels = tkinter.Label(testGui, text = texts)
            labels.place(x=(tempButton.winfo_reqwidth()),y=(25*i) + 50)
            names.append(labels)
            removeButton = tkinter.Button(testGui, text = "\u2612", command = lambda t=i, label=labels: removecheck(t,label))
            removeButton.config(bd=0)
            kill.append(removeButton)
            removeButton.place(x=(labels.winfo_reqwidth() + tempButton.winfo_reqwidth()), y=(25*i) + 50)
            i += 1
            snakePort = tkinter.IntVar()
            inputport.delete(0,tkinter.END)
            snakeName = ""
            inputsnake.delete(0,tkinter.END)
            if len(snakelist) > 6:
                testGui.minsize(width=200,height=(testGui.winfo_reqheight()+25))
    except ValueError:
        testGui.focus()
    testGui.focus()
def testcheck(j):
        snake = snakelist[j]
        if check[j]["text"] == "\u2610":
            check[j].config(text="\u2611")
            snakesRun[snake] = True
        else:
            check[j].config(text="\u2610")
            snakesRun[snake] = False
def removecheck(t, label):
        global i
        snake = snakelist[t]
        label3 = kill[t]
        checktemp = check[t]
        nametemp = names[t]
        #print(len(snakeDict))
        if kill[t]["text"] == "\u2612":
            if len(snakeDict) > t + 1:
                """snaketemps = snakelist[(t+1)]
                snakelist[t] = snaketemps
                #tempkey = snakeDict.pop(snaketemps)
                #tempvalue = snakeDict[snaketemps]
                tempvalue = snaketemps
                #print(tempkey)
                print(tempvalue)
                tempdel = kill[t+1]
                checkdel = check[t+1]
                namesdel = names[t+1]
                snakelist.remove(snaketemps)
                del snakesRun[snake]
                del snakeDict[snake]
                names[t+1].destroy()
                kill[t+1].destroy()
                check[t+1].destroy()"""
                """names.remove(nametemp)
                check.remove(checktemp)
                kill.remove(label3)"""
                """names[t] = namesdel
                check[t] = checkdel
                kill[t] = tempdel
                snakelist[t] = snaketemps
                i -= 1"""
                w = []
                w.append(names[t].place_info()['y'])
                for m in range((len(snakeDict)-(t+1))):
                    #print("eee" + str(t) + 'E' + str((t+1+m)))
                    tempx = (check[t+1+m].place_info()['x'])
                    tempy = (check[t+m].place_info()['y'])
                    temp2x = (kill[t+1+m].place_info()['x'])
                    temp3x = (names[t+1+m].place_info()['x'])
                    #print(tempy)
                    tempy = w[0]
                    w.remove(tempy)
                    w.append(names[t+1+m].place_info()['y'])
                    kill[t+1+m].config(command = lambda n=t+m: removecheck(n,label))
                    check[t+1+m].config(command = lambda j=t+m: testcheck(j))
                    check[t+1+m].place(x=tempx,y=tempy)
                    kill[t+1+m].place(x=temp2x,y=tempy)
                    names[t+1+m].place(x=temp3x,y=tempy)
                snakelist.remove(snake)
                del snakesRun[snake]
                del snakeDict[snake]
                names[t].place_forget()
                kill[t].place_forget()
                check[t].place_forget()
                names.remove(nametemp)
                check.remove(checktemp)
                kill.remove(label3)
                
                """print(snakelist)
                print(snakeDict)
                print(kill)"""
                i -= 1
            else:
                snakelist.remove(snake)
                del snakesRun[snake]
                del snakeDict[snake]
                names[t].destroy()
                kill[t].destroy()
                check[t].destroy()
                names.remove(nametemp)
                check.remove(checktemp)
                kill.remove(label3)
                #kill[t].config(command = lambda t=t: removecheck(t,label))
                """print(snakelist)
                print(snakeDict)
                print(kill)"""
                i -= 1
            if len(snakelist) >= 6:
                testGui.minsize(width=200,height=(testGui.winfo_reqheight()-25))
"""for snakes in snakelist:
    tempstr = str(i)
    if i < 100:
        tempstr = "0" + tempstr
    if i < 10:
        tempstr = "0" + tempstr 
    snakeDict[snakes] = "http://0.0.0.0:8" + tempstr
    snakesRun[snakes] = False
    tempButton = tkinter.Button(testGui, text = "\u2610", command = lambda j=i: testcheck(j))
    tempButton.place(x=0, y=(25*i) + 50)
    tempButton.config(bd=0)
    check.append(tempButton)
    labels = tkinter.Label(testGui, text = snakes)
    labels.place(x=(tempButton.winfo_reqwidth()),y=(25*i) + 50)
    i += 1"""
def test_run():
    temp = 'battlesnake play -W 11 -H 11 '
    for snakes in snakelist:
        if snakesRun[snakes]:
            temp += ' --name ' + snakes + ' --url ' + snakeDict[snakes]
    temp += ' --browser'
    os.system(temp)
    #webview.create_window()
    #webview.start()
run = tkinter.Button(testGui, text = "Run", command = test_run,)
run.place(x=0,y=0)
inputsnake = tkinter.Entry(testGui, textvariable=snakeName)
inputsnake.place(x=(run.winfo_reqwidth()),y=10)
inputsnake.place(x=(testGui.winfo_reqwidth()-inputsnake.winfo_reqwidth()),y=(run.winfo_reqheight()+1))
inputport = tkinter.Entry(testGui, textvariable=snakePort)
inputport.config(width=4)
inputport.place(x=(testGui.winfo_reqwidth()-inputport.winfo_reqwidth()),y=(run.winfo_reqheight()+22))
inputport.delete(0,tkinter.END)
label1 = tkinter.Label(testGui, text="Port:")
label1.place(x=(testGui.winfo_reqwidth()-(inputport.winfo_reqwidth() + label1.winfo_reqwidth())),y=(run.winfo_reqheight()+22))
label2 = tkinter.Label(testGui, text="Snake Name:")
label2.place(x=(testGui.winfo_reqwidth()-(inputsnake.winfo_reqwidth() + label2.winfo_reqwidth())),y=(run.winfo_reqheight()+1))
temptest = tkinter.Button(testGui,text="Submit", command=lambda texts = inputsnake.get(), ports = inputport.get(): addsnake(texts,ports))
temptest.place(x=(testGui.winfo_reqwidth()-temptest.winfo_reqwidth()),y=0)
#print(testGui.winfo_reqwidth())
#testGui.resizable(width=False, height=True)
testGui.maxsize(width=170, height=720)
testGui.minsize(width=200, height=200)
testGui.mainloop()