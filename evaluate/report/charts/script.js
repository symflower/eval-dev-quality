import { loadCSVFile, modelsWithCostsAndScores } from "./data.js";
import { evaluationTable, evaluationScatterPlot } from "./charts.js";

const evaluationRecords = await loadCSVFile("evaluation.csv");
const metaInformationRecords = await loadCSVFile("meta.csv");

const modelsWithCostsAndScoresRecords = modelsWithCostsAndScores(
    evaluationRecords,
    metaInformationRecords
);

evaluationTable(evaluationRecords);
evaluationScatterPlot(modelsWithCostsAndScoresRecords);
