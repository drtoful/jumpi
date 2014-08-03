function Terminal(lines, columns) {
    spaces = ""; for(i=0;i<columns;i++) { spaces += " "; }

    this.lines = [];
    for(i=0;i<lines;i++) { this.lines[this.lines.length] = spaces; }
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

    setTimeout(function() {
        self._draw();
        self._index += 1;
        self._update(self);
    }, data.delay / 1000);
}

Terminal.prototype.load = function(data) {
    this._data = data;
    this._index = 0
}

Terminal.prototype.attach = function(selector) {
    this.terminal = $(selector);
    this._draw();
}

Terminal.prototype.run = function() {
    this._update(this);
}
