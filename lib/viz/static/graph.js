/**
 * Shows the selected endpoint's graph and hides all others
 * @param {string} endpoint - The endpoint identifier to display
 */
function showEndpoint(endpoint) {
  document.querySelectorAll(".endpoint-graph").forEach((el) => {
    el.classList.remove("active");
  });

  if (endpoint) {
    document.getElementById(endpoint).classList.add("active");
  }
}

/**
 * Updates the graph to show only the specified number of latest points
 * @param {string|number} limit - Number of points to show, or "0" for all points
 */
function updatePointLimit(limit) {
  const activeGraph = document.querySelector(".endpoint-graph.active");
  if (!activeGraph) return;

  const limitNum = parseInt(limit);
  const allPoints = Array.from(activeGraph.querySelectorAll(".point-group"));
  const totalPoints = allPoints.length;

  const startIndex = limitNum === 0 ? 0 : Math.max(0, totalPoints - limitNum);
  const visiblePoints = allPoints.slice(startIndex);

  const graphWidth = 1000;
  const spacing = graphWidth / Math.max(1, visiblePoints.length - 1);

  updatePositions(visiblePoints, 50, spacing);

  allPoints.slice(0, startIndex).forEach((point) => {
    point.style.display = "none";
    const label = activeGraph.querySelector(
      `.label-group[data-index="${point.dataset.index}"]`
    );
    if (label) label.style.display = "none";
  });

  const path = activeGraph.querySelector(".connection-line");
  if (visiblePoints.length > 0) {
    path.setAttribute("d", generatePathData(visiblePoints));
  }
}

/**
 * Updates the position and visibility of graph elements
 * @param {Array<Element>} points - Array of point elements to reposition
 * @param {number} startX - Starting X coordinate
 * @param {number} spacing - Space between points
 */
const updatePositions = (points, startX, spacing) => {
  points.forEach((point, i) => {
    const x = startX + i * spacing;
    point.style.display = "";
    const circle = point.querySelector("circle");
    circle.setAttribute("cx", x);

    const label = activeGraph.querySelector(
      `.label-group[data-index="${point.dataset.index}"]`
    );
    if (label) {
      label.style.display = "";
      label.querySelector("text").setAttribute("x", x);
    }
  });
};

/**
 * Updates the connection line path between visible points
 * @param {Array<Element>} points - Array of visible point elements
 * @returns {string} SVG path data string
 */
const generatePathData = (points) => {
  return points
    .map((p, i) => {
      const circle = p.querySelector("circle");
      return (
        (i === 0 ? "M" : "L") +
        " " +
        circle.getAttribute("cx") +
        " " +
        circle.getAttribute("cy")
      );
    })
    .join(" ");
};

/**
 * Initializes the graph display when the page loads
 * - Selects the first endpoint by default
 * - Sets default point limit to 20
 */
window.onload = function () {
  const select = document.getElementById("endpointSelect");
  if (select.options.length > 1) {
    select.selectedIndex = 1;
    showEndpoint(select.value);
  }
  updatePointLimit(20);
};
