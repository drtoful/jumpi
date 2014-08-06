var _timeouts = [];

function Terminal(lines, columns) {
    spaces = ""; for(i=0;i<columns;i++) { spaces += " "; }

    this.lines = [];
    for(i=0;i<lines;i++) { this.lines[this.lines.length] = spaces; }

    // set new terminal to paused
    this.running = false;

    // prepare controls
    this._controls = $("<div class=\"terminal-controls\"/>");
    this._controls.append($("<p class=\"terminal-line\" style=\"height: 1px;\"/>").html(spaces));

    var terminal = this;
    var button = $("<button class=\"btn btn-xs terminal-button\" type=\"button\" />");
    button.html("<span class=\"fa fa-play fa-fw\"></span>");
    button.click(function() {
        classes = "fa fa-fw";
        if(!terminal.running) {
            classes = classes + " fa-pause";
            setTimeout(function() { terminal.run(); }, 500);
        } else {
            classes = classes + " fa-play";
            terminal.pause();
        }
        button.html("<span class=\""+classes+"\"></span>");
    });
    this._controls.append(button);


    //this._elapsed = $("<span class=\"terminal-elapsed\" />").html(" 00:00");
    //this._controls.append(this._elapsed);
}

Terminal.prototype._draw = function() {
    this.terminal.html(""); // clear terminal

    terminal = this.terminal;
    $.each(this.lines, function(index, data) {
        span = $('<p class=\"terminal-line\" />').html(data);
        terminal.append(span);
    });
}

Terminal.prototype._update = function(self) {
    if(self._index >= self._data.recording.length) {
        return;
    }

    var data = self._data.recording[self._index];
    for(i=0;i<data.changes.length;i++) {
        self.lines[data.changes[i]] = data.data[i];

        // at least 1 character needs to be drawn
        if(self.lines[data.changes[i]] == "") {
            self.lines[data.changes[i]] = " ";
        }
    }

    if(self.running) {
        _timeouts.push(setTimeout(function() {
            self._draw();
            self._index += 1;
            self._update(self);
        }, data.delay / 1000));
    }
}

Terminal.prototype.load = function(data) {
    this._data = data;
    this._index = 0
}

Terminal.prototype.attach = function(selector) {
    this.terminal = $(selector);

    // cleanup old mess
    $(".terminal-controls").remove(); // remove old terminal controls?
    for(i=0;i<_timeouts.length;i++) {
        clearTimeout(_timeouts[i]); // clear old timeouts
    }
    _timeouts = [];

    // add new controls and draw empty frame
    this.terminal.after(this._controls);
    this._draw();
}

Terminal.prototype.run = function() {
    this.running = true;
    this._update(this);
}

Terminal.prototype.pause = function() {
    this.running = false;
}
