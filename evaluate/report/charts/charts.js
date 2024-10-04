// evaluationTable creates a table given an evaluation CSV file.
function evaluationTable(data) {
    const table = $("#evaluation-table");
    const columns = data.columns;

    // Create the header.
    let thead = "<thead><tr>";
    columns.forEach((col) => {
        thead += `<th>${col}</th>`;
    });
    thead += "</tr></thead>";
    table.append(thead);

    // Create the body.
    let tbody = "<tbody>";
    data.forEach((row) => {
        tbody += "<tr>";
        columns.forEach((col) => {
            tbody += `<td>${row[col]}</td>`;
        });
        tbody += "</tr>";
    });
    tbody += "</tbody>";
    table.append(tbody);

    // Initialize the DataTable.
    table.DataTable({
        order: [[0, "asc"]],
        dom: "lfrtip",
    });
}

// evaluationScatterPlot creates a scatter plot comparing models costs and scores.
function evaluationScatterPlot(data) {
    // Set dimensions and margins for the scatter plot.
    const margin = { top: 20, right: 30, bottom: 50, left: 70 },
        width = 800,
        height = 600;

    // Append the SVG element to the body.
    const svg = d3
        .select("#evaluation-scatter")
        .append("svg")
        .attr("width", width + margin.left + margin.right)
        .attr("height", height + margin.top + margin.bottom)
        .append("g")
        .attr("transform", `translate(${margin.left},${margin.top})`);

    // Set up the x and y scales.
    const x = d3
        .scaleLinear()
        .domain([0, d3.max(data, (d) => d["model-costs"]) * 1.1])
        .range([0, width]);

    const y = d3
        .scaleLinear()
        .domain([
            d3.min(data, (d) => d.score),
            d3.max(data, (d) => d.score * 1.1),
        ])
        .range([height, 0]);

    // Add the grid lines for the x-axis.
    const xAxisGrid = d3.axisBottom(x).tickSize(-height).tickFormat("");
    svg.append("g")
        .attr("class", "grid")
        .attr("transform", `translate(0, ${height})`)
        .call(xAxisGrid);

    // Add the grid lines for the y-axis.
    const yAxisGrid = d3.axisLeft(y).tickSize(-width).tickFormat("");
    svg.append("g").attr("class", "grid").call(yAxisGrid);

    // Add the x-axis.
    svg.append("g")
        .attr("transform", `translate(0,${height})`)
        .call(d3.axisBottom(x))
        .append("text")
        .attr("x", width / 2)
        .attr("y", margin.bottom - 10)
        .attr("fill", "black")
        .attr("text-anchor", "middle")
        .style("font-size", "16px")
        .text("Costs per token");

    // Add the y-axis.
    svg.append("g")
        .call(d3.axisLeft(y))
        .append("text")
        .attr("transform", "rotate(-90)")
        .attr("x", -height / 2)
        .attr("y", -margin.left + 15)
        .attr("fill", "black")
        .attr("text-anchor", "middle")
        .style("font-size", "16px")
        .text("Score");

    // Add points.
    svg.append("g")
        .selectAll("circle")
        .data(data)
        .enter()
        .append("circle")
        .attr("cx", (d) => x(d["model-costs"]))
        .attr("cy", (d) => y(d.score))
        .attr("r", 5)
        .attr("fill", "green");

    // Add labels.
    svg.append("g")
        .selectAll("text")
        .data(data)
        .enter()
        .append("text")
        .attr("x", (d) => x(d["model-costs"]))
        .attr("y", (d) => y(d.score))
        .attr("dy", -10)
        .attr("text-anchor", "middle")
        .text((d) => d["model-name"])
        .attr("font-size", "16px")
        .attr("fill", "black");
}

export { evaluationTable, evaluationScatterPlot };
